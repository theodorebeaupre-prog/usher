<div align="center">

<img src="assets/banner/usher-banner.gif" alt="usher — un petit placeur en pixels entre en scène, éclaire les lettres u-s-h-e-r avec sa lampe de poche, et la devise s'écrit : right this way." width="640">

### Une seule commande. Le bon agent IA. À tous les coups.

Vous payez déjà pour Claude Code, Codex, Gemini CLI, Copilot… **usher décide lequel prend la tâche** — selon le type de travail, les forces de chaque agent, et qui a encore du quota. Bâti sur les abonnements que vous avez déjà. **Aucune clé API.**

[![release](https://img.shields.io/github/v/release/theodorebeaupre-prog/usher?color=4c1)](https://github.com/theodorebeaupre-prog/usher/releases/latest)
[![CI](https://github.com/theodorebeaupre-prog/usher/actions/workflows/ci.yml/badge.svg)](https://github.com/theodorebeaupre-prog/usher/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MIT-8a8a8a)](LICENSE)
[![made for](https://img.shields.io/badge/made_for-your_terminal-56b6c2)](#comment-ça-marche)

🇬🇧 [English version](README.md)

</div>

> **v0.3 — fonctionnalités complètes avant le gel 1.0.** Chaque exemple de terminal ci-dessous est un comportement réel, pas une promesse. Un accroc ? [Les issues sont bienvenues](https://github.com/theodorebeaupre-prog/usher/issues).

---

## Pourquoi

Si vous prenez le codage assisté par IA au sérieux en 2026, vous payez probablement deux ou trois abonnements d'agents en même temps. Chacun a sa propre CLI, sa personnalité, ses forces — et ses limites d'usage que vous découvrez en pleine tâche, au pire moment possible.

Chaque tâche commence donc par le même rituel silencieux : *quel agent j'ouvre pour ça ?* Et chaque limite atteinte finit par le même : *bon, je retape tout dans l'autre.*

usher élimine le rituel. Vous décrivez la tâche ; il vous installe au bon siège.

```console
$ usher "fix the flaky auth test in photocull"
→ claude  (debug task · quota OK · override with --agent)
```

…et vous voilà dans une session Claude Code normale, dans votre dépôt, l'invite déjà livrée. usher est un **lanceur, pas un intermédiaire** — il vous remet la vraie interface de l'agent et s'efface. Rien ne s'interpose entre vous et l'outil que vous payez.

## Démarrage rapide

```console
$ brew install theodorebeaupre-prog/tap/usher              # macOS
$ go install github.com/theodorebeaupre-prog/usher@latest  # partout où Go existe
```

Ou prenez un [binaire précompilé](https://github.com/theodorebeaupre-prog/usher/releases/latest) (macOS / Linux / Windows, amd64 + arm64).

Ensuite, faites l'appel, et allez-y :

```console
$ usher doctor
$ usher "explique ce que fait ce dépôt"
```

## Comment ça marche

Trois choses se passent dans les millisecondes avant la passation — tout en local, sans appel réseau, même hors ligne :

1. **Détection** — trouve les CLI d'agents installées (Claude Code, Codex, Gemini CLI, opencode, Amp, GitHub Copilot CLI, Cursor, Qwen Code).
2. **Classement** — une heuristique transparente note chaque agent : classification du type de tâche (`debug / feature / refactor / review / docs / test / other`) × poids de forces par agent × pénalité de confiance-quota × vos règles épinglées.
3. **Passation** — vous remet la session interactive du gagnant. Plein TTY, expérience native, zéro latence ajoutée là où ça compte.

Pas certain de comprendre son choix ? Demandez-lui.

```console
$ usher --why "review this PR for security issues"
task type: review

  agent       strength   quota    pin   score
  codex           0.95    1.00      —    0.95  ← launching
  gemini          0.70    1.00      —    0.70
  opencode        0.70    1.00      —    0.70
  claude          0.85    0.71      —    0.60

→ codex  (review task · quota OK · override with --agent)
```

(Ce `0.71`, c'est claude qui refroidit après une limite récente — meilleur relecteur après codex, mais usher contourne le mur au lieu de foncer dedans.)

### D'où viennent les scores

usher ne *sait* pas quel agent est le meilleur — il a **des opinions écrites noir sur blanc, multipliées par ce qu'il a observé.** Trois ingrédients :

1. **Le type de tâche** — une simple correspondance de mots-clés classe votre phrase dans une catégorie, vérifiée par ordre de priorité : *fix/crash/flaky* → `debug` avant *test* → `test`, ce qui fait que « fix the flaky test » compte comme du débogage, pas du test. Aucune correspondance → `other`.
2. **Des a priori de force** — chaque adaptateur embarque une table écrite à la main. Un extrait des vraies valeurs par défaut :

   | | debug | feature | refactor | review | docs | test |
   |---|---|---|---|---|---|---|
   | claude | **.90** | **.90** | **.95** | .85 | .85 | .85 |
   | codex | .85 | .80 | .80 | **.95** | .70 | **.90** |
   | gemini | .70 | .75 | .70 | .70 | **.90** | .70 |
   | cursor | .80 | .85 | .85 | .75 | .70 | .80 |

   En toute honnêteté : ce sont des **jugements éditoriaux sur la réputation de chaque outil, pas des benchmarks** — le refactoring multi-fichiers de Claude Code mérite son `.95` de la même façon que le mode revue de Codex mérite le sien. Ce sont des a priori : délibérément visibles, délibérément modifiables par vous.
3. **La confiance-quota** — chaque force est multipliée par ce que le [registre](#contourner-les-limites-dusage) a constaté. Une limite observée il y a 40 minutes réduit cet agent jusqu'à ce que sa fenêtre se rétablisse : le meilleur *disponible* bat le meilleur *sur papier*.

Les épingles battent les scores, `--agent` bat tout, et `--why` montre le calcul au complet. usher peut se tromper — il ne peut jamais être mystérieux.

Pourquoi pas un LLM dans la boucle ? Le routage coûterait alors de la latence, du réseau et du quota — *dépenser du quota pour décider comment dépenser du quota* — et vous ne pourriez ni le prédire ni le contredire. Le défaut reste ennuyeux exprès ; le plan à long terme, c'est que **vous l'entraîniez** : chaque décision de routage qui vous déplaît est un poids à ajuster dans votre config, et la passe 1.0 raffinera les valeurs par défaut à partir de l'usage réel.

Et quand vous n'êtes pas d'accord sur le moment, vous gagnez : `usher --agent claude "…"` saute le classement au complet, et le [fichier de config](#configuration) rend votre opinion permanente.

## Contourner les limites d'usage

Les fournisseurs n'exposent pas le quota restant, alors usher ne prétend pas le connaître. Il tient plutôt un petit registre local d'événements **observés** : chaque lancement, et chaque session qui se termine sur une erreur de limite. La fenêtre de chaque fournisseur (les ~5 heures glissantes de Claude, et compagnie) fait décroître la pénalité vers zéro.

C'est un signal de confiance, pas de la comptabilité — documenté honnêtement comme tel. En pratique, c'est la différence entre usher et un alias de shell :

```console
$ usher "add dark mode to the settings pane"
→ claude  (feature task · codex is capped — routed around it · override with --agent)
```

Limite atteinte *en pleine session* ? Quand l'agent se ferme, usher le remarque, l'enregistre, et vous offre le second choix. Votre invite voyage avec vous ; le rituel du retapage est mort.

## Scripts et CI

Mode « headless » : `-p` exécute le mode réponse-et-sortie de l'agent gagnant — la réponse atterrit sur stdout, le bavardage de routage d'usher reste sur stderr, et le code de sortie de l'agent est le vôtre.

```console
$ usher -p "summarize the failing tests in one paragraph"
→ claude  (test task · headless)          # stderr
The three failures share one cause: …     # stdout — envoyez-le où vous voulez
```

Et la partie que votre job de nuit va adorer : si l'agent atteint sa limite, usher bascule automatiquement vers le meilleur suivant — chaque agent tenté au plus une fois, aucun humain requis. L'entrée standard reçue par tube est mise en tampon et rejouée à chaque tentative, donc l'agent de relève voit la même entrée que le premier.

```console
$ usher -p "fix the crash"
→ claude  (debug task · headless)
→ claude hit its cap — failing over to codex
Patched the nil-check in auth.go; tests pass.
```

Pour l'outillage, `--json` enveloppe l'exécution dans un seul objet lisible par machine (la seule chose imprimée sur stdout — agent final, type de tâche, code de sortie, réponse, et le détail de chaque tentative), et `--timeout` empêche un agent gelé de bloquer votre pipeline : tout le groupe de processus est tué, sortie 124, convention GNU.

```console
$ usher -p --json --timeout 5m "fix the crash" | jq .agent
"codex"
```

Le timeout s'applique par tentative : une chaîne de bascules peut donc prendre jusqu'à N× la valeur. En cas d'échec total, le `exit_code` de l'enveloppe reflète le code de sortie du processus (la dernière tentative peut afficher `-1` si elle n'a pas démarré).

## Configuration

Aucune requise — usher fonctionne tel quel. Quand vous voulez des opinions, elles tiennent dans un seul fichier TOML :

```toml
# ~/.config/usher/config.toml — chaque clé est optionnelle

default_agent = "claude"        # gagne les égalités
disabled = ["opencode"]         # jamais routé ici

[weights.codex]                 # modifiez n'importe quelle force, par agent × type
review = 0.99                   # types : debug feature refactor review docs test other

[pins.types]                    # ce type de tâche va TOUJOURS à cet agent
review = "codex"

[pins.paths]                    # le travail sous ce dossier va à cet agent
"/Users/vous/travail/monorepo" = "claude"   # le préfixe correspondant le plus long gagne
```

Les épingles battent les scores ; `--agent` bat tout. Respecte `XDG_CONFIG_HOME` ; sous Windows, le fichier vit dans `%AppData%\usher`.

## L'état de la salle

`usher doctor` montre ce qui est installé, la confiance-quota de chaque agent, et où vivent votre config et votre registre :

```console
$ usher doctor
  claude     2.1.201 (Claude Code) quota ██████████ 100%
  codex      codex-cli 0.142.0 quota ██████████ 100%
  gemini     0.47.0         quota ██████████ 100%
  opencode   1.17.18        quota ██████████ 100%
  amp        not installed → npm install -g @sourcegraph/amp
  copilot    not installed → npm install -g @github/copilot
  cursor     not installed → curl https://cursor.com/install -fsS | bash
  qwen       not installed → npm install -g @qwen-code/qwen-code
  config: ~/.config/usher/config.toml (not found — using defaults)
  ledger: ~/.config/usher/ledger.json (2 events, confidence not accounting)
```

## Agents pris en charge

| Agent | Abonnement utilisé | Adaptateur |
|---|---|---|
| **Claude Code** | Claude Pro / Max | ✅ v0.1 |
| **Codex** | ChatGPT Plus / Pro | ✅ v0.1 |
| **Gemini CLI** | Google AI Pro / palier gratuit | ✅ v0.1 |
| **opencode** | à vous de choisir (abonnements inclus) | ✅ v0.1 |
| **Amp** | Amp gratuit / Pro | ✅ v0.3 |
| **GitHub Copilot CLI** | Copilot Free / Pro / Pro+ | ✅ v0.2 |
| **Cursor CLI** | Cursor Hobby / Pro | ✅ v0.2 |
| **Qwen Code** | palier gratuit Qwen / ModelScope | ✅ v0.3 |
| *le vôtre ?* | | [un adaptateur, c'est un fichier](#contribuer) |

## FAQ

**Comment je connecte mon compte Claude / ChatGPT / Copilot ?**
Vous ne le connectez pas — il n'y a rien à connecter. usher exécute les mêmes commandes que vous utilisez déjà (`claude`, `codex`, …), avec les connexions qu'elles ont déjà. Si un agent fonctionne quand vous tapez son nom, usher peut router vers lui ; `usher doctor` vous montre qui est dans la salle.

**C'est correct vis-à-vis des fournisseurs ?**
usher ne touche jamais à une API, un jeton ou une connexion. Il démarre la même CLI officielle que vous démarreriez vous-même — votre installation, votre authentification, leur interface — exactement comme un alias de shell avec du jugement. Si vous avez le droit de taper `claude`, vous avez le droit qu'on le tape pour vous.

**usher téléphone-t-il à la maison ?**
Non. Zéro appel réseau, zéro télémétrie. La détection est une recherche dans le PATH, le routage est de l'arithmétique locale, et le registre est un fichier JSON dans `~/.config/usher/` que vous pouvez lire (ou supprimer) quand vous voulez.

**Où est l'IA dans le routage ?**
Il n'y en a pas — des heuristiques de mots-clés et votre config, déterministes et explicables avec `--why`. C'est voulu : le routage n'ajoute aucune latence, fonctionne hors ligne, et ne brûle jamais de quota pour décider comment dépenser du quota.

**La « confiance-quota », c'est fiable à quel point ?**
Elle est bâtie sur ce qu'usher a *vu* — lancements et erreurs de limite observées, avec décroissance sur la fenêtre de chaque fournisseur. Elle est honnête sur sa nature : un signal, pas de la comptabilité — les fournisseurs n'exposent les vrais chiffres à personne.

**Mon agent n'est pas pris en charge.**
[Un seul fichier](CONTRIBUTING.md). Détection, arguments de lancement, motifs d'erreur de limite, profil de forces — copiez un adaptateur existant et envoyez la PR.

## Principes de conception

- **Un lanceur, pas un intermédiaire.** usher n'analyse, ne filtre et ne rebadge jamais la sortie d'un agent. Vous obtenez l'outil natif, toujours.
- **Vos abonnements, pas des clés API.** L'économie que vous avez déjà choisie continue de fonctionner.
- **Transparent plutôt que malin.** Chaque décision de routage est explicable (`--why`), contournable (`--agent`) et configurable (TOML). Pas de magie, pas de LLM dans le chemin de routage.
- **Une fiabilité ennuyeuse.** Un seul binaire Go statique. Pas de démon, pas de processus d'arrière-plan, rien à mettre à jour quand les fournisseurs retouchent leurs modèles.

## Feuille de route

- [x] Spécification — [la lire](docs/superpowers/specs/2026-07-11-usher-design.md) *(en anglais)*
- [x] Identité : logotype + [bannière animée de terminal](assets/banner/) (ingénierie inspirée de [la bannière du CLI GitHub Copilot](https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/))
- [x] v0.1 — 4 adaptateurs, routeur heuristique, registre de quota, `doctor`, `--why`
- [x] v0.2 — mode headless (`-p`) avec bascule automatique, adaptateurs Copilot + Cursor
- [x] v0.3 — enveloppe `--json`, garde `--timeout`, rejeu de stdin, adaptateurs Qwen + Amp
- [ ] v1.0 — gel de l'interface, complétions shell, page man, adaptateurs vérifiés sur de vraies machines
- [ ] Plus tard — routage assisté par LLM (optionnel), multi-comptes

## Contribuer

Toute l'architecture converge vers une seule contribution : **ajouter un agent, c'est écrire un fichier.** Un adaptateur déclare comment détecter la CLI, comment la lancer (interactif et headless), comment reconnaître son erreur de limite, et son profil de forces. C'est toute l'interface — voir [CONTRIBUTING.md](CONTRIBUTING.md) *(en anglais)*.

Des adaptateurs qu'on aimerait recevoir en PR : **Aider**, **Goose**, **Droid** — ou n'importe quelle CLI que vous payez et qui manque au tableau. Un message d'erreur de limite capturé dans une vraie session vaut de l'or (toutes nos fixtures de quota sauf une sont des reconstructions — aidez-nous à les remplacer par du vrai).

## Voici l'usher

<img src="assets/usher-orange.png" alt="logotype usher, variante orange en pixels" width="360">

Il n'a qu'un seul travail : vous mener au bon siège. Il le prend au sérieux — nœud papillon rouge compris. Regardez-le travailler — lancez usher sans arguments :

```console
$ usher
```

*(Respecte `NO_COLOR` et `USHER_NO_BANNER` ; imprime une image fixe quand la sortie est redirigée ; jamais envoyé aux lecteurs d'écran. Les images sont générées par [frames.py](assets/banner/frames.py) — les mêmes données pilotent la bannière Go et le GIF ci-dessus.)*

## Licence

[MIT](LICENSE) © 2026 ISO NORD CA

<div align="center">
<sub>Si usher vous a épargné un retapage aujourd'hui, une ⭐ aide la prochaine personne à trouver la porte.</sub>
<br><br>
<sub>usher — <em>right this way.</em></sub>
</div>
