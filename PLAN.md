# Plan de réalisation — git-purge

Découpage en 8 phases, pensées pour être exécutées **séquentiellement** par un agent de codage.
Chaque phase = un lot de commits cohérent, testable indépendamment avant de passer à la suivante.

---

## Phase 0 — Scaffolding & fondations
**Objectif** : structure de projet compilable, vide mais fonctionnelle.

- Initialiser le module Go (`go mod init github.com/<user>/git-purge`)
- Créer l'arborescence :
  ```
  cmd/git-purge/main.go
  pkg/git/
  pkg/analyzer/
  pkg/safety/
  pkg/ui/
  ```
- Ajouter les dépendances de base : `cobra`, `bubbletea`, `lipgloss`, `go-git` ou wrapper `exec.Command`
- `main.go` qui lance juste `git-purge --help`
- Config `golangci-lint`, `Makefile` (build, test, lint, run)
- Premier commit + repo GitHub initialisé

**Skills agent nécessaires** : Go tooling, structuration de repo, configuration CI de base.

---

## Phase 1 — Git Engine Adapter (`pkg/git`)
**Objectif** : couche d'abstraction fiable au-dessus de Git.

Fonctions à exposer :
- `ListLocalBranches() ([]Branch, error)`
- `ListRemoteTrackingStatus(branch) (gone bool, err error)`
- `MergeBase(a, b string) (string, error)`
- `DiffTree(commit string) (string, error)`
- `CurrentBranch() (string, error)`
- `DeleteBranch(name string, force bool) error`
- `ReflogSHA(branch string) (string, error)`

Décision technique à trancher tôt : `exec.Command("git", ...)` (simple, fiable, dépend du binaire git) vs `go-git` (pur Go, mais réimplémente moins bien certaines subtilités type diff-tree). **Recommandation : exec.Command**, plus proche du comportement réel de git.

**Skills** : parsing de sortie git (`git for-each-ref`, `git branch -vv`, `git diff-tree`), gestion d'erreurs process, tests avec dépôts git temporaires (`t.TempDir()` + `git init`).

---

## Phase 2 — Branch Analyzer (`pkg/analyzer`)
**Objectif** : classifier chaque branche locale.

Catégories :
- `MERGED` (fast-forward / merge classique détecté par `git branch --merged`)
- `SQUASH_MERGED` (algorithme décrit dans le README : merge-base + diff-tree + comparaison avec l'état de `main`)
- `GONE` (upstream supprimé, `[gone]` détecté)
- `PROTECTED` (main, master, dev, patterns custom)
- `ACTIVE_UNMERGED` (à conserver, rien à faire)

Implémenter l'algo squash-merge :
1. `merge-base(main, branch)`
2. `diff-tree` de branch vs merge-base
3. Comparer ce diff (ou l'état résultant des fichiers) à l'historique de `main` après le merge-base
4. Si identique → `SQUASH_MERGED`

**Skills** : algorithmique de comparaison d'arbres git, tests unitaires avec scénarios de squash-merge simulés (créer un vrai petit repo de test avec squash réel via `git merge --squash`).

---

## Phase 3 — Safety & Execution Engine (`pkg/safety`)
**Objectif** : garantir qu'aucune perte de données n'est possible.

- Filtre des branches protégées (liste par défaut + patterns custom `--protect`)
- Refus de supprimer la branche courante (`HEAD`)
- Enregistrement systématique du SHA avant suppression (fichier log local, ex. `.git-purge/reflog-<timestamp>.log`) pour permettre un `git reflog` / `git branch <name> <sha>` de récupération
- Vérification de working tree propre si nécessaire
- Mode `--dry-run` : simule sans exécuter `git branch -D`
- Mode `--keep-days N` : exclut les branches modifiées récemment

**Skills** : sécurité applicative (fail-safe design), logging, tests de non-régression sur la protection.

---

## Phase 4 — CLI Layer (Cobra)
**Objectif** : toutes les commandes/flags exposés proprement.

Voir la section "Routes CLI" plus bas pour le détail commande par commande.

**Skills** : Cobra (commands, flags persistants vs locaux), UX de CLI (messages clairs, exit codes cohérents).

---

## Phase 5 — TUI Layer (Bubbletea + Lipgloss)
**Objectif** : mode interactif par défaut.

- Liste des branches avec statut coloré (Merged / Squash-merged / Gone / Protected / Active)
- Sélection multiple (space), tout sélectionner (a), inverser (i)
- Écran de confirmation récapitulatif avant suppression
- Affichage du SHA de reflog après suppression
- Barre d'aide (touches disponibles) en bas d'écran

**Skills** : Bubbletea (Model/Update/View), Lipgloss (styles, tableaux), tests de composants TUI (via `teatest` si possible).

---

## Phase 6 — Tests & CI
**Objectif** : fiabilité du projet.

- Tests unitaires par package (`pkg/git`, `pkg/analyzer`, `pkg/safety`) avec dépôts git temporaires réels
- Tests d'intégration bout-en-bout (scénario complet : créer branches, merger, squash-merger, lancer git-purge --dry-run puis réel, vérifier résultat)
- GitHub Actions : lint (`golangci-lint`), test (`go test ./...`), build multi-OS (linux/mac/windows)
- Coverage minimum à définir (ex. 75%)

**Skills** : CI GitHub Actions, tests d'intégration git, matrices de build cross-platform.

---

## Phase 7 — Packaging & Release
**Objectif** : binaires installables facilement.

- `goreleaser` configuré pour macOS/Linux/Windows (amd64 + arm64)
- Formule Homebrew (optionnel)
- Page Releases GitHub avec changelog automatique
- Mise à jour du README (badges, liens de download réels)

**Skills** : goreleaser, Homebrew tap, semantic versioning / conventional commits.

---

## Phase 8 — Polish & Edge Cases
**Objectif** : robustesse finale.

- Gestion des dépôts sans remote configuré
- Gestion des branches avec caractères spéciaux
- Gros repos (perf : parallélisation des `diff-tree` si nécessaire)
- Messages d'erreur explicites (droits insuffisants, pas dans un repo git, etc.)
- Documentation `--help` complète, exemples dans le README

---

## Routes CLI (commandes & flags)

| Commande / usage | Description |
|---|---|
| `git-purge` | Mode interactif (TUI), défaut |
| `git-purge --dry-run` | Simule sans supprimer |
| `git-purge --force --yes` | Mode non-interactif, purge automatique des candidats sûrs |
| `git-purge --keep-days 14` | Exclut les branches modifiées dans les N derniers jours |
| `git-purge --protect "main,master,release/*,staging"` | Patterns de branches protégées |
| `git-purge list` | Liste les branches classifiées sans agir |
| `git-purge version` | Affiche la version |

Flags persistants (disponibles sur toutes les commandes) : `--dry-run`, `--protect`, `--keep-days`, `--verbose`.

## Graphe de dépendances entre packages

```
cmd/git-purge  →  pkg/ui  →  pkg/analyzer  →  pkg/git
                     ↓             ↓
                pkg/safety ←───────┘
```

`pkg/git` ne dépend de rien d'autre dans le projet (couche la plus basse). `pkg/safety` dépend de `pkg/git` uniquement. `pkg/analyzer` dépend de `pkg/git`. `pkg/ui` et `cmd` orchestrent le tout.

## Skills globaux nécessaires pour l'agent

- Go (idiomatique, gestion d'erreurs, tests table-driven)
- Git interne (plumbing commands, merge-base, diff-tree, reflog)
- Cobra (CLI)
- Bubbletea + Lipgloss (TUI)
- Tests avec dépôts git réels temporaires
- CI/CD GitHub Actions
- goreleaser / packaging cross-platform