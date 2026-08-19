package pepin.rules

import rego.v1

# ── CLD-K8S-4 : RBAC, cluster-admin au-delà du binding natif ──

_crb(name, role, subjects) := {"resources": [{
	"provider": "kubernetes", "type": "k8s_cluster_role_binding", "id": name,
	"attributes": {"name": name, "role_ref": role, "subjects": subjects},
}]}

# ✓ Le binding livré avec Kubernetes lie cluster-admin à system:masters : attendu, pas un écart.
test_k8s_builtin_masters_binding_ok if {
	count({f | some f in deny; f.code == "k8s_rbac_no_cluster_admin_binding"}) == 0 with input as _crb(
		"cluster-admin", "cluster-admin",
		[{"kind": "Group", "name": "system:masters"}],
	)
}

# ✗ Tout autre sujet recevant cluster-admin est une délégation de pleins pouvoirs.
test_k8s_extra_cluster_admin_denied if {
	some f in deny with input as _crb("admin", "cluster-admin", [{"kind": "User", "name": "alice"}])
	f.code == "k8s_rbac_no_cluster_admin_binding"
}

# ✗ Un ServiceAccount avec cluster-admin compte aussi (chemin d'attaque classique).
test_k8s_serviceaccount_cluster_admin_denied if {
	some f in deny with input as _crb("ci", "cluster-admin", [{"kind": "ServiceAccount", "name": "ci-runner"}])
	f.code == "k8s_rbac_no_cluster_admin_binding"
}

# ✓ Un binding vers un rôle restreint n'est pas concerné.
test_k8s_non_admin_role_ok if {
	count({f | some f in deny; f.code == "k8s_rbac_no_cluster_admin_binding"}) == 0 with input as _crb("viewer", "view", [{"kind": "User", "name": "bob"}])
}

# ── CLD-K8S-5 : Pod Security Standards ──

_ns(name, labels) := {"resources": [{
	"provider": "kubernetes", "type": "k8s_namespace", "id": name,
	"attributes": {"name": name, "labels": labels},
}]}

# ✗ Label absent : aucun garde-fou.
test_k8s_pss_missing_denied if {
	some f in deny with input as _ns("app", {})
	f.code == "k8s_namespace_pod_security_enforced"
}

# ✗ RÉGRESSION : `privileged` n'impose AUCUNE restriction — c'est le pire cas, il doit
# échouer comme l'absence de label. Ne tester que la présence le laissait passer au vert.
test_k8s_pss_privileged_denied if {
	some f in deny with input as _ns("app", {"pod-security.kubernetes.io/enforce": "privileged"})
	f.code == "k8s_namespace_pod_security_enforced"
}

# ✓ baseline et restricted imposent bien un garde-fou.
test_k8s_pss_baseline_ok if {
	count({f | some f in deny; f.code == "k8s_namespace_pod_security_enforced"}) == 0 with input as _ns("app", {"pod-security.kubernetes.io/enforce": "baseline"})
}

test_k8s_pss_restricted_ok if {
	count({f | some f in deny; f.code == "k8s_namespace_pod_security_enforced"}) == 0 with input as _ns("app", {"pod-security.kubernetes.io/enforce": "restricted"})
}

# ✓ Les namespaces de plateforme sont exclus : leur durcissement n'appartient pas au client.
test_k8s_pss_platform_namespace_exempt if {
	count({f | some f in deny; f.code == "k8s_namespace_pod_security_enforced"}) == 0 with input as _ns("kube-system", {})
}

# ── CLD-K8S-6 : NetworkPolicy ──

# ✗ Aucune NetworkPolicy dans le namespace : réseau plat.
test_k8s_no_networkpolicy_denied if {
	some f in deny with input as _ns("app", {"pod-security.kubernetes.io/enforce": "restricted"})
	f.code == "k8s_namespace_network_policy_defined"
}

# ✓ Une NetworkPolicy dans CE namespace suffit à lever l'écart.
test_k8s_networkpolicy_present_ok if {
	count({f | some f in deny; f.code == "k8s_namespace_network_policy_defined"}) == 0 with input as {"resources": [
		{"provider": "kubernetes", "type": "k8s_namespace", "id": "app", "attributes": {"name": "app", "labels": {"pod-security.kubernetes.io/enforce": "restricted"}}},
		{"provider": "kubernetes", "type": "k8s_network_policy", "id": "np", "attributes": {"name": "np", "namespace": "app"}},
	]}
}

# ✗ Une NetworkPolicy dans un AUTRE namespace ne protège pas celui-ci.
test_k8s_networkpolicy_other_namespace_denied if {
	some f in deny with input as {"resources": [
		{"provider": "kubernetes", "type": "k8s_namespace", "id": "app", "attributes": {"name": "app", "labels": {"pod-security.kubernetes.io/enforce": "restricted"}}},
		{"provider": "kubernetes", "type": "k8s_network_policy", "id": "np", "attributes": {"name": "np", "namespace": "autre"}},
	]}
	f.code == "k8s_namespace_network_policy_defined"
}

# ── CLD-K8S-10 : gestionnaire de secrets externes ──

_crds(names) := {"resources": [c |
	some n in names
	c := {"provider": "kubernetes", "type": "k8s_crd", "id": n, "attributes": {"name": n}}
]}

# ✗ Aucune CRD de gestionnaire de secrets : les secrets reposent sur le stockage natif.
test_k8s_no_external_secrets_denied if {
	some f in deny with input as _crds(["certificates.cert-manager.io", "ingresses.networking.k8s.io"])
	f.code == "k8s_secrets_external_manager"
}

# ✓ External Secrets Operator détecté.
test_k8s_external_secrets_operator_ok if {
	count({f | some f in deny; f.code == "k8s_secrets_external_manager"}) == 0 with input as _crds(["externalsecrets.external-secrets.io"])
}

# ✓ Secrets Store CSI Driver détecté.
test_k8s_secrets_store_csi_ok if {
	count({f | some f in deny; f.code == "k8s_secrets_external_manager"}) == 0 with input as _crds(["secretproviderclasses.secrets-store.csi.x-k8s.io"])
}

# ✓ Sans AUCUNE CRD collectée, on ne conclut pas (la collecte n'a pas eu lieu).
test_k8s_secrets_no_crd_collected_silent if {
	count({f | some f in deny; f.code == "k8s_secrets_external_manager"}) == 0 with input as {"resources": []}
}
