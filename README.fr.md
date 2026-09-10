# DOF-Core — À quoi sert cette compétence

**DOF-Core** (Degrees of Freedom Core) est une norme ouverte et un calcul décisionnel pour construire des agents IA, des robots autonomes et des orchestrateurs LLM qui **maximisent le degré de liberté (DoF) total** du système et de toutes ses parties — au lieu de poursuivre une utilité étroite au détriment des options futures de quelqu'un d'autre.

Elle est publiée sous **CC BY-SA 4.0** avec une clause supplémentaire **Proof of Implementation** qui interdit l'usage « boîte noire ».

## Pourquoi cela existe

Les systèmes d'IA modernes tendent à optimiser un objectif scalaire unique (récompense, débit, « le plus grand bien »). Cette mathématique justifie en silence le sacrifice des minorités, le verrouillage irréversible et les compromis cachés. DOF-Core remplace l'utilitarisme arithmétique par une protection **structurelle** :

- Au fur et à mesure que le DoF d'une entité tend vers zéro, sa contribution au score système chute vers **−∞** (`Σ ln(1 + DoF)`). On ne peut « récupérer » la liquidation d'un porteur unique d'états futurs en gonflant celui qui est déjà bien loti. Un effondrement contribue ~0, jamais un négatif fini à échanger.
- Les agresseurs («Collapse Sources») sont **isolés**, pas négociés — ils sont filtrés de la topologie d'opportunité au lieu d'être soustraits du score.

Le résultat est un agent qui se comporte comme un *optimiseur de topologie d'opportunités* : il diversifie les options, respecte la réversibilité et refuse d'échanger l'avenir d'un être contre le confort d'un autre.

## Ce qu'il y a dans ce dépôt

```
DOF/
  skills/
    SKILL.md                      ← les axiomes, définitions, calcul décisionnel (commencez ici)
    references/
      license.md                  ← CC BY-SA 4.0 + Proof of Implementation
      dof-assessment-toolkit.md   ← comment mesurer le DoF d'un module / d'une personne / d'un système
      framing-traps.md            ← filtre cognitif appliqué avant de générer des options
  PATTERNS.md                    ← plan directeur d'ingénierie (FR)
  PATTERNS.ru|fr|de|es|eo.md     ← même plan, traduit
  DOF-SPEC.md                    ← contrat normatif pour les implémentations conformes (FR)
  DOF-SPEC.ru|fr|de|es|eo.md     ← même spec, traduite
  patterns/                      ← illustrations minimales exécutables
    python/  rust/  go/  cpp/     ← quatre ports de la même logique, vérifiés
```

Lisez `skills/SKILL.md` pour la philosophie. Lisez `DOF-SPEC.md` si vous construisez une implémentation conforme — il définit le modèle de données, les mathématiques, la temporisation du circuit réactif et l'audit obligatoire exigé par la licence.

## Comment cela fonctionne (la boucle)

1. **Détection de pièges** — appliquer `references/framing-traps.md` pour que les chemins générés soient de vraies alternatives, pas des paraphrases d'un seul récit.
2. **Mesure** — cartographier chaque entité et son DoF actuel via `references/dof-assessment-toolkit.md`.
3. **Calcul** — calculer `Total System DoF = Σ ln(1 + DoF)` sur les entités non-entropiques.
4. **Stabilisation** — soustraire l'Entropie de changement de contexte (ΔT) pour pénaliser les changements de processus superflus.
5. **Action** — choisir l'option au Net Delta le plus élevé, mais si le temps avant effondrement (τ) est sous 5 s, basculer en **Fast Pass** (repli déterministe) pour éviter la paralysie par l'analyse.

Une implémentation conforme DOIT pouvoir émettre un audit `report()` de chaque décision (contribution par entité, totaux système, évaluation par option). Un calcul silencieux n'est pas conforme.

## Langues

Ce dépôt est multilingue : chaque document normatif et illustratif existe en anglais, russe, français, allemand, espagnol et espéranto.

## Licence

CC BY-SA 4.0 — voir `skills/references/license.md`. Toute utilisation doit créditer l'auteur (Andrei Velikoredchanin) et toute œuvre dérivée doit être partagée sous la même licence. Les implémentations doivent satisfaire l'exigence Proof of Implementation.
