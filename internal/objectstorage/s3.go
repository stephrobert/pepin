// Package objectstorage est le collecteur de stockage objet S3-compatible,
// PARTAGÉ par tous les providers souverains (Outscale OOS, Scaleway Object
// Storage…). Le stockage objet de ces clouds expose l'API S3 ; seuls l'endpoint
// et les identifiants changent. Il projette chaque bucket dans le type normalisé
// agnostique `object_storage_bucket`, audité par les mêmes règles quel que soit
// le provider.
package objectstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/stephrobert/pepin/internal/model"
)

// CollectBuckets liste les buckets d'un endpoint S3-compatible et projette
// chacun dans le modèle normalisé. `provider` étiquette les ressources ;
// `endpoint` vide = endpoint S3 par défaut (sinon endpoint souverain custom).
// `sseKMS` indique que le provider expose le chiffrement par défaut du bucket via
// une clé client (SSE-KMS) : seul un tel provider renseigne `sse_kms_enabled`
// (CLD-CHF-4) ; les autres laissent l'attribut absent (capacité inexistante).
func CollectBuckets(ctx context.Context, provider, endpoint, region, accessKey, secretKey string, sseKMS bool) ([]model.Resource, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("configuration S3 : %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = &endpoint
		}
		o.UsePathStyle = true // requis par les endpoints S3-compatibles souverains.
	})

	listed, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("ListBuckets : %w", err)
	}
	var out []model.Resource
	for _, b := range listed.Buckets {
		if b.Name == nil {
			continue
		}
		out = append(out, collectBucket(ctx, client, provider, region, *b.Name, sseKMS))
	}
	return out, nil
}

// collectBucket interroge ACL, versioning, policy et tags d'un bucket (best
// effort : un appel non supporté/absent n'interrompt pas la collecte).
func collectBucket(ctx context.Context, client *s3.Client, provider, region, name string, sseKMS bool) model.Resource {
	attrs := map[string]any{"name": name}

	if acl, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{Bucket: &name}); err == nil {
		attrs["acl_grants"] = grantsToMap(acl.Grants)
		attrs["public_via_acl"] = isACLPublic(acl.Grants)
	}
	if v, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: &name}); err == nil {
		attrs["versioning"] = string(v.Status)
	}
	if p, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: &name}); err == nil && p.Policy != nil {
		attrs["policy_public"] = policyAllowsPublic(*p.Policy)
	}
	// Un bucket SANS AUCUN tag renvoie l'erreur NoSuchTagSet : la traiter comme un échec
	// laissait l'attribut absent, donc la garde de capacité de governance_resource_required_tags
	// rendait le bucket conforme. Résultat absurde : un bucket avec 1 tag sur 4 était flagué,
	// un bucket avec ZÉRO tag ne l'était pas. « Aucun tag » est une valeur, pas une erreur.
	t, tagErr := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: &name})
	switch {
	case tagErr == nil:
		tags := make([]any, 0, len(t.TagSet))
		for _, tag := range t.TagSet {
			tags = append(tags, map[string]any{"key": deref(tag.Key), "value": deref(tag.Value)})
		}
		attrs["tags"] = tags
	case isEmptyTagSet(tagErr):
		attrs["tags"] = []any{}
	}
	// Object Lock (immutabilité/WORM, CLD-STO-8). On distingue TROIS cas pour ne pas produire
	// un faux FAIL : verrou lu (valeur réelle), « jamais configuré » (NotFound ⇒ non activé),
	// ou ERREUR de collecte (403/timeout ⇒ attribut ABSENT ⇒ contrôle non-évalué, pas un FAIL).
	if ol, err := client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{Bucket: &name}); err == nil {
		attrs["object_lock_enabled"] = ol.ObjectLockConfiguration != nil &&
			ol.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled
	} else if isNotConfigured(err) {
		attrs["object_lock_enabled"] = false
	} // sinon : attribut non renseigné → garde de capacité → not-evaluated.

	// Chiffrement par défaut du bucket. DEUX notions distinctes, à ne pas confondre :
	//   - `default_encryption_enabled` : le bucket chiffre-t-il au repos, quel que soit
	//     l'algorithme (CLD-CHF-2). Le SSE est OPT-IN par bucket chez plusieurs providers
	//     souverains : un bucket peut donc parfaitement ne rien chiffrer.
	//   - `sse_kms_enabled` : la clé est-elle gérée par le CLIENT (BYOK, CLD-CHF-4) —
	//     capacité que tous les providers n'ont pas, d'où l'attribut posé seulement si
	//     `sseKMS`. Les confondre revenait à déclarer le chiffrement « non applicable »
	//     alors que son activation, elle, est bien observable.
	enc, encErr := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: &name})
	switch {
	case encErr == nil:
		rules := 0
		if enc.ServerSideEncryptionConfiguration != nil {
			rules = len(enc.ServerSideEncryptionConfiguration.Rules)
		}
		attrs["default_encryption_enabled"] = rules > 0
		if sseKMS {
			attrs["sse_kms_enabled"] = false
			for _, rule := range enc.ServerSideEncryptionConfiguration.Rules {
				d := rule.ApplyServerSideEncryptionByDefault
				if d != nil && d.SSEAlgorithm == types.ServerSideEncryptionAwsKms {
					attrs["sse_kms_enabled"] = true
					attrs["kms_key_id"] = deref(d.KMSMasterKeyID)
				}
			}
		}
	case isNotConfigured(encErr):
		// « Aucune configuration de chiffrement » est une VALEUR : le bucket ne chiffre pas.
		attrs["default_encryption_enabled"] = false
		if sseKMS {
			attrs["sse_kms_enabled"] = false
		}
	} // sinon (403, timeout) : attributs absents → not-evaluated, jamais un faux vert.

	// Chaque attribut n'est posé QUE si son appel a réussi (voir ci-dessus) : attester
	// l'appel qui le porte est donc une lecture fidèle de ce qui s'est passé, jamais une
	// recopie de ce que le code déclare appeler. Un 403 sur GetBucketAcl laisse l'attribut
	// absent ET non attesté — l'attestation ne peut pas nommer un appel qui a échoué.
	return model.Resource{
		Provider: provider, Type: "object_storage_bucket", ID: name, Name: name, Region: region,
		Attributes: attrs,
		Provenance: model.AttestPresent(attrs, bucketAttrCall, bucketAttrDerived),
	}
}

// bucketAttrCall : l'appel S3 qui porte chaque attribut de bucket.
var bucketAttrCall = map[string]string{
	"name":                       "s3:ListBuckets",
	"acl_grants":                 "s3:GetBucketAcl",
	"public_via_acl":             "s3:GetBucketAcl",
	"versioning":                 "s3:GetBucketVersioning",
	"policy_public":              "s3:GetBucketPolicy",
	"tags":                       "s3:GetBucketTagging",
	"object_lock_enabled":        "s3:GetObjectLockConfiguration",
	"default_encryption_enabled": "s3:GetBucketEncryption",
	"sse_kms_enabled":            "s3:GetBucketEncryption",
	"kms_key_id":                 "s3:GetBucketEncryption",
}

// bucketAttrDerived : les attributs CALCULÉS depuis la réponse (un booléen déduit
// d'une liste de grants n'est pas un champ de l'API), par opposition à ceux qui en
// sont recopiés.
var bucketAttrDerived = map[string]bool{
	"public_via_acl":             true,
	"policy_public":              true,
	"object_lock_enabled":        true,
	"default_encryption_enabled": true,
	"sse_kms_enabled":            true,
}

// isACLPublic — un grant cible le groupe public (AllUsers/AuthenticatedUsers).
func isACLPublic(grants []types.Grant) bool {
	for _, g := range grants {
		if g.Grantee != nil && g.Grantee.URI != nil {
			uri := *g.Grantee.URI
			if strings.Contains(uri, "AllUsers") || strings.Contains(uri, "AuthenticatedUsers") {
				return true
			}
		}
	}
	return false
}

// grantsToMap projette les grants ACL S3 en [{grantee:{uri,id,type}, permission}],
// forme attendue par les règles `object_storage_bucket`.
func grantsToMap(grants []types.Grant) []any {
	out := make([]any, 0, len(grants))
	for _, g := range grants {
		grantee := map[string]any{}
		if g.Grantee != nil {
			grantee["uri"] = deref(g.Grantee.URI)
			grantee["id"] = deref(g.Grantee.ID)
			grantee["type"] = string(g.Grantee.Type)
		}
		out = append(out, map[string]any{"grantee": grantee, "permission": string(g.Permission)})
	}
	return out
}

// policyAllowsPublic analyse la policy de bucket (JSON) et détecte une exposition publique :
// un Statement `Effect: Allow` avec un Principal (ou NotPrincipal) joker « * », SANS condition
// restreignant réellement la source (SourceIp/Vpc/PrincipalOrgID…). Le parsing structuré
// remplace l'ancien matching de chaînes, fragile au formatage et aveugle aux conditions.
func policyAllowsPublic(policy string) bool {
	var doc struct {
		Statement json.RawMessage `json:"Statement"`
	}
	if json.Unmarshal([]byte(policy), &doc) != nil || len(doc.Statement) == 0 {
		return false
	}
	stmts := statementList(doc.Statement)
	for _, st := range stmts {
		if statementIsPublic(st) {
			return true
		}
	}
	return false
}

// statementList normalise le champ Statement (objet unique OU tableau) en une liste.
func statementList(raw json.RawMessage) []map[string]json.RawMessage {
	var arr []map[string]json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return arr
	}
	var one map[string]json.RawMessage
	if json.Unmarshal(raw, &one) == nil {
		return []map[string]json.RawMessage{one}
	}
	return nil
}

// statementIsPublic : Effect=Allow + Principal/NotPrincipal joker, sans condition restreignant
// la source (une condition SecureTransport seule ne « déprivatise » pas — le bucket reste public).
func statementIsPublic(st map[string]json.RawMessage) bool {
	var effect string
	if json.Unmarshal(st["Effect"], &effect) != nil || !strings.EqualFold(effect, "Allow") {
		return false
	}
	if conditionRestrictsSource(st["Condition"]) {
		return false
	}
	return principalIsWildcard(st["Principal"]) || principalIsWildcard(st["NotPrincipal"])
}

// principalIsWildcard reconnaît « * » sous toutes ses formes : "*", {"AWS":"*"}, {"AWS":["*"]}.
func principalIsWildcard(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "*"
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	for _, v := range obj {
		var one string
		if json.Unmarshal(v, &one) == nil && one == "*" {
			return true
		}
		var arr []string
		if json.Unmarshal(v, &arr) == nil {
			for _, x := range arr {
				if x == "*" {
					return true
				}
			}
		}
	}
	return false
}

// conditionRestrictsSource : vrai si la condition borne réellement l'origine de l'appel
// (IP, VPC, organisation, compte, principal). Les autres conditions (ex. SecureTransport)
// n'empêchent pas l'accès public et ne comptent donc pas comme restriction.
func conditionRestrictsSource(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	restrictKeys := []string{
		"aws:sourceip", "aws:sourcevpc", "aws:sourcevpce", "aws:vpcsourceip",
		"aws:principalorgid", "aws:principalorgpaths", "aws:principalaccount",
		"aws:principalarn", "aws:sourceaccount", "aws:sourcearn", "aws:sourceowner",
	}
	lower := strings.ToLower(string(raw))
	// Un opérateur NÉGATIF inverse le sens : `NotIpAddress`/`StringNotEquals` autorise
	// TOUT SAUF l'exception citée — donc la policy est PLUS ouverte, pas restreinte. La
	// classer « restreinte » sur la seule présence de la clé rendait un bucket public
	// invisible. En présence d'une négation, on refuse de conclure à une restriction.
	for _, neg := range []string{"notipaddress", "stringnotequals", "stringnotlike", "arnnotequals", "arnnotlike", "notaction", "notresource", "notprincipal"} {
		if strings.Contains(lower, neg) {
			return false
		}
	}
	for _, k := range restrictKeys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// isEmptyTagSet : « ce bucket n'a aucun tag » est une réponse, pas un échec de collecte.
func isEmptyTagSet(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NoSuchTagSet")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// isNotConfigured distingue « la fonctionnalité n'a jamais été configurée sur ce bucket »
// (réponse LÉGITIME de l'API : le verrou/chiffrement est simplement absent = non activé) d'une
// erreur de collecte (403/timeout). Seul le premier cas autorise à conclure « non activé » ;
// une erreur réelle laisse l'attribut absent, pour que le contrôle soit non-évalué, pas en échec.
func isNotConfigured(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NotFoundError") ||
		strings.Contains(msg, "ConfigurationNotFound") ||
		strings.Contains(msg, "ObjectLockConfigurationNotFound") ||
		strings.Contains(msg, "NoSuchObjectLockConfiguration") ||
		strings.Contains(msg, "ServerSideEncryptionConfigurationNotFound")
}
