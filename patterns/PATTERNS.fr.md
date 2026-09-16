# Modèles d'Implémentation DOF-Core

Ce document décrit l'implémentation technique du cadre DOF-Core pour l'intégration dans les systèmes d'IA, les robots autonomes et les orchestrateurs LLM. L'objectif est de séparer la créativité générative de la validation mathématique stricte.

---

## 🇫🇷 Français: Modèles d'Implémentation

### Modèle 1 : Architecture Isolée à Trois Couches
Le système est divisé en trois contours isolés avec un flux de données unidirectionnel pour empêcher les hallucinations de l'IA d'affecter les actions physiques.

1. **Couche de Perception et de Cartographie (Graph Mapper) :**
   - **Tâche :** Analyse l'environnement et construit un `StateGraph`. Traduit les objets physiques en structures `Entity` avec un vecteur DoF numérique. Calcule le $\tau$ global (Time-to-Collapse). Lit les **moyens de l'agent agissant par ressource**, les groupes d'échange dérivés et leurs taux observés (§3.2, §4.8) — les blocs de la lentille Options sont **dérivés** des exigences, de ces moyens et des groupes, jamais rédigés à la main (§4.6).
2. **Couche de Synthèse (Le Générateur) :**
   - **Tâche :** Reçoit le graphe. Génère un ensemble de stratégies hypothétiques (3 à 5 chemins distincts). Le contrôle direct des actionneurs lui est interdit.
3. **Couche de Validation (Calculus Core) :**
   - **Tâche :** Reçoit les plans du Générateur. Simule chaque option. Les filtre via la formule non linéaire $\sum \ln(\text{DoF})$. Une option qui conduit une entité *comptée* à un zéro connu porte une **charge d'effondrement** et est **retirée de l'ensemble des candidats** tant qu'une alternative sans charge existe (admissibilité structurelle, `DOF-SPEC` §4.2/§4.5) ; dans l'audit, un effondrement apparaît comme le plancher fini $\ln \varepsilon$, jamais comme un nombre qu'un gain ailleurs pourrait racheter. Une option que l'agent **ne peut pas payer** est retirée de la même manière : comparaison directe avec les moyens de l'agent, puis une conversion *vérifiée* à l'intérieur d'un groupe d'échange au taux observé (le temps propre de l'échange est imputé au même $\tau$), et si la pénurie y survit, l'option porte `gate = "insolvency"` (§4.8) — non payable est un verdict, pas un prix.

### Modèle 2 : Cercle Réactif avec Interruption
Évite la « paralysie par l'analyse » en liant les cycles de calcul au temps physique restant avant l'effondrement ($\tau$).

- **Si $\tau \ge 5000000.0$ µs (5 secondes) :** **Diversification Profonde**. Activation de la couche LLM pour rechercher des alternatives cachées (3 à 5 options distinctes).
- **Si $\tau < 5000000.0$ µs (5 secondes) :** **Passage Rapide (Fast Pass)**. Le Générateur est court-circuité. Le système passe à l'unique scénario déterministe prédéfini (Minimax Bounds).

### Modèle 3 : Tuyau d'Évaluation du Calcul (Calculus Evaluator Pipe)
Implémentation déterministe (Python/Rust/C++) du noyau d'évaluation.
- **Logique :** Calcul du DoF global du système.
- **Sélection :** $\text{Net Delta} = \text{Total System DoF Evaluation Index}_{\text{projected}} - \text{Total System DoF Evaluation Index}_{\text{current}} - \Delta T$.
- **Contrainte :** Les actions irréversibles reçoivent une pénalité structurelle (ex: $-0.5$).
- **Porte des ressources :** chaque option déclare ce qu'elle prélève sur l'agent agissant (§3.3 ; négatif = consommation, `energy` écrit explicitement, même `0.0`). Une option est **retirée, non pénalisée**, si le prélèvement dépasse les moyens déclarés même après conversion vérifiée complète (`gate = "insolvency"`, §4.8) ; chaque retrait figure dans `removed_options`.
- **Référence :** ne rien faire est la ligne de base — aucun $\Delta T$, donc $\text{Net Delta} = 0$ par définition. Une option n'est sélectionnée que si son $\text{Net Delta}$ est **strictement positif** ; sinon le système reste sur place et l'audit l'enregistre.

### Implémentation de Référence (Fichiers de Code)
Le répertoire `patterns/` contient un SDK Python exécutable implémentant toutes les couches :
- `patterns/python/graph_mapper.py` — Couche de Perception et de Cartographie (construit `SystemStateMatrix`, calcule τ).
- `patterns/python/generator.py` — Couche de Synthèse (génération d'options via LLM avec repli déterministe).
- `patterns/python/calculus_core.py` — Couche de Validation (somme logarithmique DoF, sélection via ΔT).
- `patterns/python/orchestrator.py` — Cercle Réactif avec Interruption (lie les couches ; FAST PASS / DEEP DIVERSIFICATION).
- `patterns/python/smoke_test.py` — Exemple minimal exécutable.
- **Portages multilingues** (même logique, exécutables vérifiés) :
  - `patterns/rust/` — portage Rust (`dof_core.rs`, `graph_mapper.rs`, `generator.rs`, `orchestrator.rs`, `main.rs`).
  - `patterns/go/` — portage Go (`dof_core.go`, `graph_mapper.go`, `generator.go`, `orchestrator.go`, `main.go`, `go.mod`).
  - `patterns/cpp/` — portage C++ (`dof_core.hpp`, `graph_mapper.hpp`, `generator.hpp`, `orchestrator.hpp`, `main.cpp`).
- `patterns/tools/verify_ports.sh` — exécute les quatre ports contre une empreinte figée et affiche le nombre de vérifications de chacun ; la preuve de conformité du §7 en une commande.
