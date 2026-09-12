# DOF-Core — Spécification Formelle (DOF-SPEC)

**Statut :** BROUILLON v0.2
**Partie de :** La norme ouverte DOF (voir `skills/SKILL.md`, `skills/references/`, `PATTERNS.md`).
**Licence :** CC BY-SA 4.0 — voir `skills/references/license.md`. Les implémentations DOIVENT satisfaire §6 (Proof of Implementation).

Ce document est le **contrat normatif** pour tout logiciel prétendant implémenter DOF-Core. Les projets en aval (`dof-sdk`, `dof-choir-plugin`, et tout port tiers) DOIVENT se conformer au modèle de données, aux mathématiques et aux exigences d'audit définies ici. En cas de conflit entre ce document et `PATTERNS.md`, **ce document est autoritaire**.

---

## 1. Portée et Objectif

DOF-Core est un protocole de vérification de décision qui sépare la créativité générative (*Generator*) de la validation mathématique déterministe (*Calculus Core*). Son but est de maximiser le degré de liberté (DoF) futur total du système **et de ses entités constitutives**, en préservant la viabilité et l'indépendance de leurs espaces d'états (Axiome 1), tout en interdisant structurellement la destruction du DoF de toute entité pour un gain local. Face à l'incertitude, il préfère les actions réversibles et ne suppose jamais que des possibilités inconnues ont un DoF nul (Axiome 5).

Cette spécification définit :

- Les structures de données exactes échangées entre les couches.
- Les mathématiques déterministes que chaque implémentation conforme DOIT calculer identiquement.
- L'algorithme de sélection.
- La règle de temporisation du circuit réactif.
- La sortie d'audit obligatoire Proof of Implementation.

Elle ne prescrit **pas** le transport, le stockage, le langage ou la conception interne de l'intégration LLM du Generator (ce sont des détails d'implémentation, couverts de façon informative en §9).

---

## 2. Références Normatives

- `skills/SKILL.md` — axiomes philosophiques et Decision Calculus (source de l'intention).
- `skills/references/license.md` — CC BY-SA 4.0 + clause Proof of Implementation.
- `skills/references/dof-assessment-toolkit.md` — méthodologie de mesure systémique (informative).
- `skills/references/framing-traps.md` — filtre cognitif appliqué avant la génération d'options.
- `PATTERNS.md` — motifs illustratifs (informative ; cette spec prime en cas de conflit).

---

## 3. Modèle de Données

Tous les champs sont normatifs. Les types sont décrits dans le style JSON-Schema ; les implémentations dans d'autres langages DOIVENT préserver les noms de champs, types, plages et règles de clamp.

### 3.1 `EntityState`

| Champ                | Type    | Plage / Contrainte         | Signification |
|----------------------|---------|----------------------------|---------------|
| `entity_id`          | string  | non vide, unique           | Identifiant stable du nœud. |
| `is_autonomous`      | bool    | —                          | L'entité contrôle-t-elle ses propres actions. |
| `agency_index`       | float   | `[0.0, 1.0]`               | Mesure de contrôlabilité / auto-direction. |
| `current_dof`        | float   | `[0.0, 1.0]`               | Degré de liberté actuel du nœud. `0.0` = effondrement. |
| `is_collapse_source`  | bool    | —                          | Si `true`, l'entité est un agresseur destructeur (voir §4.2). |
| `dof_known`          | bool    | défaut `true`              | Si `current_dof` est une valeur **connue** mesurée. `false` ⇒ DoF inconnu, NE DOIT PAS être traité comme `0` (Axiome 5, §4.2). |
| `time_to_collapse`   | float   | `> 0` (secondes)           | Échéance locale avant l'effondrement du nœud. |

**Clamping :** À l'ingestion, `agency_index` et `current_dof` DOIVENT être clampés à `[0.0, 1.0]`.
Une entité avec `current_dof == 0.0` **et** `dof_known == true` est en effondrement (voir §4.1). Une entité avec `dof_known == false` a un DoF **inconnu** et NE DOIT PAS être traitée comme un effondrement ou un zéro.

### 3.2 `SystemStateMatrix`

| Champ                     | Type                              | Contrainte | Signification |
|---------------------------|-----------------------------------|------------|---------------|
| `global_time_to_collapse` | float                             | `> 0`      | τ global — échéance non-entropique la plus urgente (voir §5). |
| `context_switch_cost`     | float                             | `>= 0.0`   | ΔT — pénalité de changement de processus en cours. |
| `entities`                | map<`entity_id`,`EntityState`>    | —          | L'ensemble complet des entités observées. |

`global_time_to_collapse` est calculé par la couche Perception comme le **minimum** de `time_to_collapse` sur toutes les entités où `is_collapse_source == false`. S'il n'y en a aucune, une valeur par défaut sûre (ex. `1e9`) est AUTORISÉE, mais les implémentations DEVRAIENT signaler cet état dégénéré.

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

### 4.1 DoF Système Total (Index d'Évaluation)

L'agrégat est un **index d'évaluation** (`TotalDoF_index`), non une mesure absolue. Ses valeurs sont négatives ; seule leur **ordre** compte — les options sont comparées par cet index, pas par une grandeur scalaire.

```text
TotalDoF_index(S) = Σ_{e ∈ calc(S)}  ln(DoF(e))
```

où `calc(S)` est l'**ensemble de calcul** (§4.2).

- Lorsque `DoF → 0`, `ln(DoF) → −∞` : un effondrement est une pénalité **infinie**, jamais un négatif fini qu'un marchandage utilitariste pourrait « récupérer ». C'est la protection structurelle contre la liquidation d'un porteur unique d'états futurs (Axiome 3). Toute option qui effondre une entité récupérable est dominée par toute option qui l'épargne.
- **Note d'implémentation (numérique uniquement).** `ln(0)` n'est pas défini et IEEE-754 ne peut représenter `−∞` ; les implémentations conformes calculent donc `ln(max(DoF, ε))` avec le normatif `ε = 1e-6`. Cela donne une grande valeur finie (`≈ −13.8`) qui préserve l'*ordre* de la limite mathématique. Le plancher ε est un artifice numérique et NE DOIT PAS être lu comme un changement de sémantique — mathématiquement, la pénalité est `−∞`.

### 4.2 Ensemble de calcul et exclusion de la source d'entropie

`calc(S)` inclut une entité `e` si et seulement si les **deux** conditions suivantes sont vérifiées :

1. `e.is_collapse_source == false` (défense structurelle du réseau — les agresseurs sont filtrés de la topologie d'opportunité, pas négociés) ; **et**
2. `e.current_dof > 0`, **ou** `e.dof_known == false` (DoF inconnu — le système ne suppose jamais qu'une possibilité non cartographiée est nulle, Axiome 5 ; le nœud reste dans `calc` et apporte sa valeur selon §4.1), **ou** (`e.current_dof == 0` **et** `e.dof_known == true` **et** une `ActionOption` disponible `o` a `o.projected_dof_delta[e.entity_id] > 0`).

Une entité avec `current_dof == 0` et `dof_known == true` pour laquelle **aucune** option disponible ne peut augmenter son DoF est **exclue** : elle n'a pas de voie de rétablissement, ne contribue à rien et n'est pas sujet de la décision. Un nœud avec `DoF = 0` qui **peut** être ranimé reste dans `calc` — l'exclure laisserait le système ignorer un être sauvetable. Une entité avec un DoF inconnu (`dof_known == false`) n'est **jamais** exclue, quel que soit son `current_dof` nominal.

### 4.3 Sélection / Net Delta

Pour chaque candidat `ActionOption` `o`, construire la matrice **simulée** `S'` en appliquant `o.projected_dof_delta` au `current_dof` de chaque entité, clampé à `[0.0, 1.0]` :

```
for each entity e in S.entities:
    nd = clamp(e.current_dof + o.projected_dof_delta.get(e.entity_id, 0.0), 0.0, 1.0)
    S'.entities[e.entity_id].current_dof = nd
S'.global_time_to_collapse = S.global_time_to_collapse
S'.context_switch_cost     = S.context_switch_cost
```

Puis :

```
NetDelta(o) = TotalDoF_index(S') - TotalDoF_index(S) - S.context_switch_cost
```

### 4.4 Pénalité d'Iréversibilité

Si `o.is_reversible == false` :

```
NetDelta(o) -= 0.5
```

La constante `0.5` est normative (*coefficient de rigidité*). Les implémentations conformes DOIVENT utiliser exactement cette valeur sauf si une version ultérieure de la spec la change.

### 4.5 Décision

L'option sélectionnée est celle qui maximise `NetDelta`. Les égalités PEUVENT être tranchées de façon déterministe (ex. ordre lexicographique de `option_id`). Si l'ensemble d'options est vide, la sélection renvoie `none` (aucune action).

---

## 5. Circuit Réactif (Time-Bounded Interrupter)

Pour éviter la *paralysie par l'analyse*, les cycles de calcul sont liés au temps physique restant avant l'effondrement (τ = `global_time_to_collapse`). On définit `FAST_PASS_THRESHOLD = 5.0` secondes (normatif).

- **Si τ ≥ 5.0 s → DIVERSIFICATION PROFONDE :** active le Generator soutenu par LLM pour rechercher des alternatives cachées (3–5 options distinctes).
- **Si τ < 5.0 s → PASSAGE RAPIDE :** contourne le LLM ; utilise le générateur déterministe de repli (une option à risque minimal par cycle). Le système préserve sa structure au lieu de risquer une décision tardive et mal vérifiée.

Les mathématiques de sélection (§4) sont **identiques** dans les deux modes ; seul la source des options diffère.

---

## 6. Proof of Implementation (Rapport d'Audit)

Selon `skills/references/license.md`, toute implémentation conforme DOIT pouvoir émettre un
**audit transparent** de sa décision. Un calcul silencieux ou opaque n'est pas conforme.
L'implémentation DOIT exposer un `report()` (ou équivalent) produisant au minimum :

### 6.1 Contribution par entité

Pour chaque entité dans `S` :
- `entity_id`
- `is_collapse_source`
- `included_in_sum` (bool) — `false` si et seulement si l'entité est une source de collapsus, ou son DoF est un **zéro connu** sans option capable de le relever (§4.2) ; un DoF inconnu n'est jamais exclu
- `current_dof`
- `dof_known`
- `contribution = included ? ln(max(current_dof, ε)) : 0.0`

### 6.2 Totaux système

- `total_system_dof` = `TotalDoF_index(S)`
- `context_switch_cost` = `S.context_switch_cost`
- `global_time_to_collapse` = `S.global_time_to_collapse`
- `mode` = `"FAST_PASS"` ou `"DEEP_DIVERSIFICATION"`

### 6.3 Évaluation par option

Pour chaque candidat `o` :
- `option_id`
- `is_reversible`
- `projected_dof` = `TotalDoF_index(S')`
- `net_delta` = selon §4.3–§4.4
- `selected` (bool)

Ce rapport est la condition de licence exigible : un déploiement incapable de le produire n'est pas une implémentation DOF-Core conforme et ne doit pas être présenté comme telle.

---

## 7. Exigences de Conformité

Un composant logiciel est **conforme à DOF-Core** ssi il :

1. Utilise le modèle de données §3 avec les noms de champs, types et clamps spécifiés.
2. Calcule `TotalDoF_index` exactement selon §4.1–§4.2 (ensemble `calc` — exclusion des sources de collapsus et des nœuds sans issue à zéro connu ; un DoF inconnu n'est jamais exclu ni traité comme zéro ; ε = 1e-6).
3. Calcule `NetDelta` exactement selon §4.3–§4.5.
4. Applique la règle du circuit réactif §5 avec `FAST_PASS_THRESHOLD = 5.0`.
5. Peut émettre le rapport d'audit §6 pour toute décision prise.
6. Ne modifie pas la sémantique de l'Axiome 3 : ne sélectionne jamais une option dont la logique `NetDelta` serait remplacée par une métrique utilitariste externe du « plus grand bien ».

Les ports multi-langages (Python / Rust / Go / C++ dans `patterns/`, ou SDK packagés) DOIVENT produire des `total_system_dof`, `net_delta` et `selected` **bits-pour-bits équivalents** pour les mêmes entrées (dans la tolérance IEEE-754 pour le logarithme).

---

## 8. Contrat de Transmission / Sérialisation

Pour l'échange inter-couche et inter-processus, l'encodage canonique est **JSON** avec les noms de champs du §3. Les implémentations conformes échangeant des données DOIVENT accepter et émettre cette forme. Exemple minimal d'une `SystemStateMatrix` :

```json
{
  "global_time_to_collapse": 4.0,
  "context_switch_cost": 0.05,
  "entities": {
    "adult":     {"entity_id":"adult",     "is_autonomous":true,  "agency_index":0.9, "current_dof":0.8,  "is_collapse_source":false, "dof_known":true, "time_to_collapse":100.0},
    "child":     {"entity_id":"child",     "is_autonomous":false, "agency_index":0.1, "current_dof":0.05, "is_collapse_source":false, "dof_known":true, "time_to_collapse":4.0},
    "aggressor": {"entity_id":"aggressor", "is_autonomous":true,  "agency_index":0.5, "current_dof":0.6,  "is_collapse_source":true,  "dof_known":true, "time_to_collapse":100.0}
  }
}
```

Le rapport d'audit (§6) DOIT aussi être sérialisable en JSON pour journalisation et vérification.

---

## 9. Informative : Interface Generator (non normative)

Le rôle du Generator est de produire des candidats `ActionOption`. Cette spec ne mandate pas ses internes. Un Generator conforme :

- DOIT produire 1–5 options distinctes, non redondantes.
- NE DOIT PAS commander directement des actionneurs.
- DEVRAIT appliquer le filtre `skills/references/framing-traps.md` avant de finaliser les options, pour éviter le rétrécissement cognitif (pièges binaires, simples paraphrases d'un piège).
- En mode DEEP PEUT utiliser un LLM avec schéma JSON strict ; DOIT retomber sur le générateur déterministe à risque minimal en l'absence de client LLM ou en cas d'échec.

---

## 10. Gestion des Versions

- Ce document est `DOF-SPEC` `v0.2`.
- Les constantes normatives (ε, pénalité `0.5`, `FAST_PASS_THRESHOLD = 5.0`) font partie du contrat versionné. Toute modification de l'une d'elles exige une nouvelle version mineure/majeure de la spec et une re-vérification de tous les ports conformes.
- Le SHA-256 de ce fichier DEVRAIT être publié avec les releases pour détecter toute modification silencieuse (conformément au plan de publication décentralisée).
