# Règles d'état DANS le cluster Kubernetes (mode --kubeconfig), complémentaires des
# contrôles de plan de contrôle managé (kubernetes_cluster_*, via l'API du fournisseur).
#   Types normalisés : k8s_cluster_role_binding (name, role_ref, subjects[]),
#   k8s_namespace (name, labels{}), k8s_network_policy (name, namespace).
# SCSL : CLD-K8S-4 (RBAC au moindre privilège), CLD-K8S-5 (Pod Security Standards),
#   CLD-K8S-6 (NetworkPolicy par défaut).
package pepin.rules

import rego.v1

# Namespaces gérés par la plateforme : le durcissement y relève du fournisseur/de
# Kubernetes lui-même, pas du client — les flaguer serait du bruit non actionnable.
_platform_ns := {"kube-system", "kube-public", "kube-node-lease"}

# ── CLD-K8S-4 : RBAC — cluster-admin accordé au-delà du binding natif ──
# Le binding livré avec Kubernetes lie cluster-admin au groupe `system:masters` : c'est
# attendu. TOUT autre sujet (utilisateur, groupe, compte de service) recevant cluster-admin
# est une délégation de pleins pouvoirs.
deny contains f if {
	some b in resources_of_type("k8s_cluster_role_binding")
	object.get(b.attributes, "role_ref", "") == "cluster-admin"
	some s in object.get(b.attributes, "subjects", [])
	not _builtin_masters(s)
	subject := sprintf("%s/%s", [object.get(s, "kind", "?"), object.get(s, "name", "?")])
	name := object.get(b.attributes, "name", b.id)
	f := {
		"code": "k8s_rbac_no_cluster_admin_binding",
		"severity": "critical",
		"subject": name,
		"message": sprintf("ClusterRoleBinding « %s » accorde cluster-admin à %s — pleins pouvoirs sur le cluster.", [name, subject]),
		"remediation": "Retirer cluster-admin de ce sujet ; accorder un Role/ClusterRole restreint au strict nécessaire.",
		"labels": {"provider": provider_of(b), "category": "security"},
	}
}

_builtin_masters(s) if {
	object.get(s, "kind", "") == "Group"
	object.get(s, "name", "") == "system:masters"
}

# ── CLD-K8S-5 : Pod Security Standards non imposés sur un namespace applicatif ──
# Le niveau `privileged` n'impose AUCUNE restriction : il est fonctionnellement équivalent
# à l'absence de label. Ne tester que la PRÉSENCE laissait donc passer le pire cas — un
# namespace explicitement ouvert aux pods privilégiés — au vert.
_pss_enforced(labels) if {
	lvl := object.get(labels, "pod-security.kubernetes.io/enforce", "")
	lvl in {"baseline", "restricted"}
}

deny contains f if {
	some n in resources_of_type("k8s_namespace")
	name := object.get(n.attributes, "name", n.id)
	not name in _platform_ns
	labels := object.get(n.attributes, "labels", {})
	not _pss_enforced(labels)
	lvl := object.get(labels, "pod-security.kubernetes.io/enforce", "(absent)")
	f := {
		"code": "k8s_namespace_pod_security_enforced",
		"severity": "high",
		"subject": name,
		"message": sprintf("Namespace « %s » sans Pod Security Standards imposés (enforce = %s) — un pod privilégié y est accepté.", [name, lvl]),
		"remediation": "Poser le label pod-security.kubernetes.io/enforce (baseline ou restricted) sur le namespace.",
		"labels": {"provider": provider_of(n), "category": "security"},
	}
}

# ── CLD-K8S-6 : aucun NetworkPolicy dans un namespace applicatif (réseau plat) ──
deny contains f if {
	some n in resources_of_type("k8s_namespace")
	name := object.get(n.attributes, "name", n.id)
	not name in _platform_ns
	not _has_netpol(name)
	f := {
		"code": "k8s_namespace_network_policy_defined",
		"severity": "high",
		"subject": name,
		"message": sprintf("Namespace « %s » sans aucune NetworkPolicy — tout pod y joint tout autre pod (réseau plat).", [name]),
		"remediation": "Définir une NetworkPolicy de refus par défaut (ingress et egress) dans le namespace, puis autoriser les flux légitimes.",
		"labels": {"provider": provider_of(n), "category": "security"},
	}
}

_has_netpol(ns) if {
	some p in resources_of_type("k8s_network_policy")
	object.get(p.attributes, "namespace", "") == ns
}

# ── CLD-K8S-10 : secrets gérés via un coffre externe ──
# Constat BINAIRE, pas une heuristique : un gestionnaire de secrets externes s'installe
# avec ses CRD (External Secrets Operator, Secrets Store CSI Driver, agent Vault). Aucune
# de ces CRD ⇒ les secrets sont gérés nativement par Kubernetes (base64 dans etcd), ce que
# l'exigence proscrit. Le finding est unique (objet identique ⇒ dédoublonné par l'ensemble).
deny contains f if {
	count(resources_of_type("k8s_crd")) > 0 # la collecte a bien eu lieu
	not _external_secret_manager
	f := {
		"code": "k8s_secrets_external_manager",
		"severity": "high",
		"subject": "cluster",
		"message": "Aucun gestionnaire de secrets externes détecté (External Secrets Operator, Secrets Store CSI, agent Vault) — les secrets reposent sur le stockage natif Kubernetes.",
		"remediation": "Déployer un gestionnaire de secrets externes (External Secrets Operator ou Secrets Store CSI) et monter les secrets depuis le coffre, jamais en clair dans les manifests.",
		"labels": {"provider": "kubernetes", "category": "security"},
	}
}

_external_secret_manager if {
	some c in resources_of_type("k8s_crd")
	name := lower(object.get(c.attributes, "name", ""))
	some marker in ["externalsecret", "secretstore", "secretproviderclass", "vault"]
	contains(name, marker)
}
