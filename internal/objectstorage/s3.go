// Package objectstorage est le collecteur de stockage objet S3-compatible,
// PARTAGÉ par tous les providers souverains (Outscale OOS, Scaleway Object
// Storage…). Le stockage objet de ces clouds expose l'API S3 ; seuls l'endpoint
// et les identifiants changent. Il projette chaque bucket dans le type normalisé
// agnostique `object_storage_bucket`, audité par les mêmes règles quel que soit
// le provider.
package objectstorage

import (
	"context"
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
// `endpoint` vide = endpoint AWS par défaut (sinon endpoint souverain custom).
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
	if t, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: &name}); err == nil {
		tags := make([]any, 0, len(t.TagSet))
		for _, tag := range t.TagSet {
			tags = append(tags, map[string]any{"key": deref(tag.Key), "value": deref(tag.Value)})
		}
		attrs["tags"] = tags
	}
	// Object Lock (immutabilité/WORM, CLD-STO-8) : absent ⇒ non activé (l'API
	// renvoie une erreur si le verrou n'a jamais été configuré sur le bucket).
	attrs["object_lock_enabled"] = false
	if ol, err := client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{Bucket: &name}); err == nil &&
		ol.ObjectLockConfiguration != nil {
		attrs["object_lock_enabled"] = ol.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled
	}

	// Clé de chiffrement gérée par le client (SSE-KMS / BYOK, CLD-CHF-4) : renseigné
	// UNIQUEMENT pour les providers exposant cette capacité (sseKMS). Le chiffrement
	// par défaut du bucket utilise une clé client quand la règle par défaut porte
	// SSEAlgorithm=aws:kms + KMSMasterKeyID (GetBucketEncryption).
	if sseKMS {
		attrs["sse_kms_enabled"] = false
		if enc, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: &name}); err == nil &&
			enc.ServerSideEncryptionConfiguration != nil {
			for _, rule := range enc.ServerSideEncryptionConfiguration.Rules {
				d := rule.ApplyServerSideEncryptionByDefault
				if d != nil && d.SSEAlgorithm == types.ServerSideEncryptionAwsKms {
					attrs["sse_kms_enabled"] = true
					attrs["kms_key_id"] = deref(d.KMSMasterKeyID)
				}
			}
		}
	}

	return model.Resource{Provider: provider, Type: "object_storage_bucket", ID: name, Name: name, Region: region, Attributes: attrs}
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

// policyAllowsPublic détecte un Statement Allow avec Principal « * » (heuristique
// alignée sur l'analyse fine déléguée aux règles Rego).
func policyAllowsPublic(policy string) bool {
	if !strings.Contains(policy, `"Allow"`) {
		return false
	}
	for _, p := range []string{
		`"Principal":"*"`, `"Principal": "*"`,
		`"Principal":{"AWS":"*"}`, `"Principal": {"AWS": "*"}`,
		`"AWS":["*"]`, `"AWS": ["*"]`,
	} {
		if strings.Contains(policy, p) {
			return true
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
