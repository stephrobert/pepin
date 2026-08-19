package objectstorage

import "errors"

import "testing"

func TestPolicyAllowsPublic(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{
			name:   "principal wildcard string",
			policy: `{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"*"}]}`,
			want:   true,
		},
		{
			name:   "principal wildcard (clé S3) objet",
			policy: `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":"s3:GetObject"}]}`,
			want:   true,
		},
		{
			name:   "principal wildcard (clé S3) tableau",
			policy: `{"Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":"s3:GetObject"}]}`,
			want:   true,
		},
		{
			name:   "single statement object (not array)",
			policy: `{"Statement":{"Effect":"Allow","Principal":"*","Action":"s3:*"}}`,
			want:   true,
		},
		{
			name:   "notprincipal wildcard is also public",
			policy: `{"Statement":[{"Effect":"Allow","NotPrincipal":{"AWS":"*"},"Action":"s3:GetObject"}]}`,
			want:   true,
		},
		{
			name:   "weird whitespace (defeats string matching, not JSON parsing)",
			policy: "{\n  \"Statement\" : [ {\n    \"Effect\"\t:\t\"Allow\",\n    \"Principal\" : \"*\"\n  } ]\n}",
			want:   true,
		},
		{
			name:   "SecureTransport-only condition is still public",
			policy: `{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Condition":{"Bool":{"aws:SecureTransport":"true"}}}]}`,
			want:   true,
		},
		{
			name:   "restricted to a source IP range is NOT public",
			policy: `{"Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Condition":{"IpAddress":{"aws:SourceIp":"203.0.113.0/24"}}}]}`,
			want:   false,
		},
		{
			name:   "restricted to an org id is NOT public",
			policy: `{"Statement":[{"Effect":"Allow","Principal":"*","Condition":{"StringEquals":{"aws:PrincipalOrgID":"o-xxxx"}}}]}`,
			want:   false,
		},
		{
			name:   "deny to everyone is NOT an exposure",
			policy: `{"Statement":[{"Effect":"Deny","Principal":"*","Action":"s3:*"}]}`,
			want:   false,
		},
		{
			name:   "specific principal is NOT public",
			policy: `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123:root"},"Action":"s3:GetObject"}]}`,
			want:   false,
		},
		{
			name:   "public statement among restricted ones is caught",
			policy: `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123:root"}},{"Effect":"Allow","Principal":"*","Action":"s3:GetObject"}]}`,
			want:   true,
		},
		{
			name:   "malformed policy does not assert public",
			policy: `not json`,
			want:   false,
		},
		{
			name:   "empty policy",
			policy: `{}`,
			want:   false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := policyAllowsPublic(c.policy); got != c.want {
				t.Errorf("policyAllowsPublic() = %v, want %v", got, c.want)
			}
		})
	}
}

// RÉGRESSION : un opérateur NÉGATIF rend la policy PLUS publique (tout Internet sauf une
// exception), or `conditionRestrictsSource` concluait « restreinte » sur la seule présence
// de la clé de condition — un bucket public devenait invisible.
func TestConditionNegativeOperatorIsNotARestriction(t *testing.T) {
	cases := map[string]string{
		"NotIpAddress":    `{"NotIpAddress":{"aws:SourceIp":["10.0.0.0/8"]}}`,
		"StringNotEquals": `{"StringNotEquals":{"aws:PrincipalAccount":["123456789012"]}}`,
		"ArnNotLike":      `{"ArnNotLike":{"aws:PrincipalArn":["arn:x"]}}`,
	}
	for name, cond := range cases {
		if conditionRestrictsSource([]byte(cond)) {
			t.Errorf("%s : classé « restreint » alors qu'il ÉLARGIT l'accès", name)
		}
	}
	// Contrôle positif : une condition positive reste bien une restriction.
	if !conditionRestrictsSource([]byte(`{"IpAddress":{"aws:SourceIp":["10.0.0.0/8"]}}`)) {
		t.Error("IpAddress positif : devrait être reconnu comme une restriction")
	}
}

// RÉGRESSION : « aucun tag » (NoSuchTagSet) est une valeur, pas une erreur de collecte.
func TestEmptyTagSetIsAValue(t *testing.T) {
	if !isEmptyTagSet(errors.New("operation error S3: GetBucketTagging, NoSuchTagSet: The TagSet does not exist")) {
		t.Error("NoSuchTagSet doit être reconnu comme « aucun tag », sinon l'attribut reste absent et le bucket passe au vert")
	}
	if isEmptyTagSet(errors.New("AccessDenied")) {
		t.Error("une vraie erreur ne doit pas être confondue avec un TagSet vide")
	}
}
