# DOF-Core — Spécification Formelle (DOF-SPEC)

**Statut :** BROUILLON v0.1
**Partie de :** La norme ouverte DOF (voir `SKILL.md`, `references/`, `PATTERNS.md`).
**Licence :** CC BY-SA 4.0 — voir `references/license.md`. Les implémentations DOIVENT satisfaire §6 (Proof of Implementation).

Ce document est le **contrat normatif** pour tout logiciel prétendant implémenter
DOF-Core. Les projets en aval (`dof-sdk`, `dof-choir-plugin`, et tout port tiers)
DOIVENT se conformer au modèle de données, aux mathématiques et aux exigences d'audit
définies ici. En cas de conflit entre ce document et `PATTERNS.md`, **ce document est autoritaire**.

---

## 1. Portée et Objectif

DOF-Core est un protocole de vérification de décision qui sépare la créativité générative
(*Generator*) de la validation mathématique déterministe (*Calculus Core*). Son but est de
maximiser le degré de liberté (DoF) total du système tout en interdisant structurellement
la destruction du DoF de toute entité pour un gain local.

Cette spécification définit :

- Les structures de données exactes échangées entre les couches.
- Les mathématiques déterministes que chaque implémentation conforme DOIT calculer identiquement.
- L'algorithme de sélection.
- La règle de temporisation du circuit réactif.
- La sortie d'audit obligatoire Proof of Implementation.

Elle ne prescrit **pas** le transport, le stockage, le langage ou la conception interne
de l'intégration LLM du Generator (ce sont des détails d'implémentation, couverts de façon
informative en §9).

---

## 2. Références Normatives

- `SKILL.md` — axiomes philosophiques et Decision Calculus (source de l'intention).
- `references/license.md` — CC BY-SA 4.0 + clause Proof of Implementation.
- `references/dof-assessment-toolkit.md` — méthodologie de mesure systémique (informative).
- `references/framing-traps.md` — filtre cognitif appliqué avant la génération d'options.
- `PATTERNS.md` — motifs illustratifs (informative ; cette spec prime en cas de conflit).

---

## 3. Modèle de Données

Tous les champs sont normatifs. Les types sont décrits dans le style JSON-Schema ; les
implémentations dans d'autres langages DOIVENT préserver les noms de champs, types, plages et règles de clamp.

### 3.1 `EntityState`

| Champ                | Type    | Plage / Contrainte         | Signification |
|----------------------|---------|----------------------------|---------------|
| `entity_id`          | string  | non vide, unique           | Identifiant stable du nœud. |
| `is_autonomous`      | bool    | —                          | L'entité contrôle-t-elle ses propres actions. |
| `agency_index`       | float   | `[0.0, 1.0]`               | Mesure de contrôlabilité / auto-direction. |
| `current_dof`        | float   | `[0.0, 1.0]`               | Degré de liberté actuel du nœud. `0.0` = effondrement. |
| `is_entropy_source`  | bool    | —                          | Si `true`, l'entité est un agresseur destructeur (voir §4.2). |
| `time_to_collapse`   | float   | `> 0` (secondes)           | Échéance locale avant l'effondrement du nœud. |

**Clamping :** À l'ingestion, `agency_index` et `current_dof` DOIVENT être clampés à `[0.0, 1.0]`.
Une entité avec `current_dof == 0.0` est en effondrement (voir §4.1).

### 3.2 `SystemStateMatrix`

| Champ                     | Type                              | Contrainte | Signification |
|---------------------------|-----------------------------------|------------|---------------|
| `global_time_to_collapse` | float                             | `> 0`      | τ global — échéance non-entropique la plus urgente (voir §5). |
| `context_switch_cost`     | float                             | `>= 0.0`   | ΔT — pénalité de changement de processus en cours. |
| `entities`                | map<`entity_id`,`EntityState`>    | —          | L'ensemble complet des entités observées. |

`global_time_to_collapse` est calculé par la couche Perception comme le **minimum**
de `time_to_collapse` sur toutes les entités où `is_entropy_source == false`. S'il n'y en a
aucune, une valeur par défaut sûre (ex. `1e9`) est AUTORISÉE, mais les implémentations
DEVRAIENT signaler cet état dégénéré.

### 3.3 `ActionOption`

| Champ                  | Type                             | Contrainte          | Signification |
|------------------------|----------------------------------|---------------------|---------------|
| `option_id`            | string                           | non vide, unique    | Identifiant stable du plan candidat. |
| `description`          | string                           | —                   | Résumé lisible. |
| `projected_dof_delta`  | map<`entity_id`, float>          | —                   | Prévision de changement de `current_dof` par entité. |
| `is_reversible`        | bool                             | —                   | `false` ⇒ irréversible ⇒ pénalité structurelle (§4.4). |

---

## 4. Mathématiques de Base

Soit ε = `1e-6` (protection contre `ln(0)`). Soit `S` la `SystemStateMatrix` courante.

### 4.1 DoF Système Total

```
TotalDoF(S) = Σ_{e ∈ S.entities, e.is_entropy_source == false}  ln(1 + max(e.current_dof, ε))
```

- La somme est prise **uniquement** sur les entités non-entropiques (voir §4.2).
- Lorsque `current_dof → 0`, `ln(1 + dof) → 0` : un effondrement contribue ~0, jamais un
  négatif fini qu'un marchandage utilitariste pourrait « récupérer ». C'est la protection
  structurelle contre la liquidation d'un porteur unique d'états futurs (Axiome 3).

### 4.2 Exclusion de la Source d'Entropie

Toute entité avec `is_entropy_source == true` est **exclue** de `TotalDoF`. Son DoF personnel
n'est pas soustrait du score système, et son isolation n'est pas pénalisée. Cela réalise la
*défense structurelle du réseau* : les agresseurs sont filtrés de la topologie d'opportunité
plutôt que négociés.

### 4.3 Sélection / Net Delta

Pour chaque candidat `ActionOption` `o`, construire la matrice **simulée** `S'` en appliquant
`o.projected_dof_delta` au `current_dof` de chaque entité, clampé à `[0.0, 1.0]` :

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Puis :

```
NetDelta(o) = TotalDoF(S') - TotalDoF(S) - S.context_switch_cost
```

### 4.4 Pénalité d'Iréversibilité

Si `o.is_reversible == false` :

```
NetDelta(o) -= 0.5
```

La constante `0.5` est normative (*coefficient de rigidité*). Les implémentations conformes
DOIVENT utiliser exactement cette valeur sauf si une version ultérieure de la spec la change.

### 4.5 Décision

L'option sélectionnée est celle qui maximise `NetDelta`. Les égalités PEUVENT être tranchées
de façon déterministe (ex. ordre lexicographique de `option_id`). Si l'ensemble d'options est
vide, la sélection renvoie `none` (aucune action).

---

## 5. Circuit Réactif (Time-Bounded Interrupter)

Pour éviter la *paralysie par l'analyse*, les cycles de calcul sont liés au temps physique
restant avant l'effondrement (τ = `global_time_to_collapse`). On définit
`FAST_PASS_THRESHOLD = 5.0` secondes (normatif).

- **Si τ ≥ 5.0 s → DIVERSIFICATION PROFONDE :** active le Generator soutenu par LLM pour
  rechercher des alternatives cachées (3–5 options distinctes).
- **Si τ < 5.0 s → PASSAGE RAPIDE :** contourne le LLM ; utilise le générateur déterministe
  de repli (une option à risque minimal par cycle). Le système préserve sa structure au lieu
  de risquer une décision tardive et mal vérifiée.

Les mathématiques de sélection (§4) sont **identiques** dans les deux modes ; seul la source
des options diffère.

---

## 6. Proof of Implementation (Rapport d'Audit)

Selon `references/license.md`, toute implémentation conforme DOIT pouvoir émettre un
**audit transparent** de sa décision. Un calcul silencieux ou opaque n'est pas conforme.
L'implémentation DOIT exposer un `report()` (ou équivalent) produisant au minimum :

### 6.1 Contribution par entité

Pour chaque entité dans `S` :
- `entity_id`
- `is_entropy_source`
- `included_in_sum` (bool) — `false` ssi `is_entropy_source`
- `current_dof`
- `contribution = included ? ln(1 + max(current_dof, ε)) : 0.0`

### 6.2 Totaux système

- `total_system_dof` = `TotalDoF(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` ou `"DEEP_DIVERSIFICATION"`

### 6.3 Évaluation par option

Pour chaque candidat `o` :
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF(S')`
- `net_delta` = selon §4.3–§4.4
- `selected` (bool)

Ce rapport est la condition de licence exigible : un déploiement incapable de le produire
n'est pas une implémentation DOF-Core conforme et ne doit pas être présenté comme telle.

---

## 7. Exigences de Conformité

Un composant logiciel est **conforme à DOF-Core** ssi il :

1. Utilise le modèle de données §3 avec les noms de champs, types et clamps spécifiés.
2. Calcule `TotalDoF` exactement selon §4.1–§4.2 (exclusion entropie ; ε = 1e-6).
3. Calcule `NetDelta` exactement selon §4.3–§4.5.
4. Applique la règle du circuit réactif §5 avec `FAST_PASS_THRESHOLD = 5.0`.
5. Peut émettre le rapport d'audit §6 pour toute décision prise.
6. Ne modifie pas la sémantique de l'Axiome 3 : ne sélectionne jamais une option dont la
   logique `NetDelta` serait remplacée par une métrique utilitariste externe du « plus grand bien ».

Les ports multi-langages (Python / Rust / Go / C++ dans `patterns/`, ou SDK packagés)
DOIVENT produire des `total_system_dof`, `net_delta` et `selected` **bits-pour-bits équivalents**
pour les mêmes entrées (dans la tolérance IEEE-754 pour le logarithme).

---

## 8. Contrat de Transmission / Sérialisation

Pour l'échange inter-couche et inter-processus, l'encodage canonique est **JSON** avec les
noms de champs du §3. Les implémentations conformes échangeant des données DOIVENT accepter
et émettre cette forme. Exemple minimal d'une `SystemStateMatrix` :

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_entropy_source":false, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_entropy_source":false, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_entropy_source":true,  "time_to_collapse":100.0}
  }
}
```

Le rapport d'audit (§6) DOIT aussi être sérialisable en JSON pour journalisation et vérification.

---

## 9. Informative : Interface Generator (non normative)

Le rôle du Generator est de produire des candidats `ActionOption`. Cette spec ne mandate pas
ses internes. Un Generator conforme :

- DOIT produire 1–5 options distinctes, non redondantes.
- NE DOIT PAS commander directement des actionneurs.
- DEVRAIT appliquer le filtre `references/framing-traps.md` avant de finaliser les options,
  pour éviter le rétrécissement cognitif (pièges binaires, simples paraphrases d'un piège).
- En mode DEEP PEUT utiliser un LLM avec schéma JSON strict ; DOIT retomber sur le
  générateur déterministe à risque minimal en l'absence de client LLM ou en cas d'échec.

---

## 10. Gestion des Versions

- Ce document est `DOF-SPEC` `v0.1`.
- Les constantes normatives (ε, pénalité `0.5`, `FAST_PASS_THRESHOLD = 5.0`) font partie du
  contrat versionné. Toute modification de l'une d'elles exige une nouvelle version mineure/majeure
  de la spec et une re-vérification de tous les ports conformes.
- Le SHA-256 de ce fichier DEVRAIT être publié avec les releases pour détecter toute
  modification silencieuse (conformément au plan de publication décentralisée).
