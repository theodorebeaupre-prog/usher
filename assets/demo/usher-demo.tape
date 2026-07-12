# usher launch demo — four scenes: the room, the brain, the failover, the bow.
# Every frame is the real v0.3.1 binary; fake agents shadow the real CLIs.

Output usher-demo.gif
Output usher-demo.mp4
Output usher-demo.webm

Set Shell bash
Set FontSize 17
Set Width 1100
Set Height 800
Set Padding 24
Set WindowBar Colorful
Set BorderRadius 10
Set Framerate 24
Set TypingSpeed 55ms
Set Theme { "name": "usher", "background": "#0d0d0d", "foreground": "#c9c9c9", "cursor": "#ffd75f", "black": "#0d0d0d", "red": "#e06c75", "green": "#98c379", "yellow": "#d7a94d", "blue": "#61afef", "magenta": "#c678dd", "cyan": "#56b6c2", "white": "#c9c9c9", "brightBlack": "#5a5a5a", "brightRed": "#ff5c5c", "brightGreen": "#6fdc8c", "brightYellow": "#ffd75f", "brightBlue": "#7cbdff", "brightMagenta": "#d69bf0", "brightCyan": "#7adfff", "brightWhite": "#f5f5f5" }

Hide
Type "source <demo-dir>/setup.sh  # see assets/demo/README.md && clear"
Enter
Show

# ── Scene 1: the room ────────────────────────────────────────────
Sleep 600ms
Type "usher doctor"
Sleep 400ms
Enter
Sleep 3400ms

Hide
Type "clear"
Enter
Show

# ── Scene 2: the brain (--why + routed-around decision) ─────────
Sleep 400ms
Type 'usher --why "review this PR for security issues"'
Sleep 400ms
Enter
Sleep 4600ms

Hide
Type "clear"
Enter
Show

# ── Scene 3: the failover (headless, cap hit, auto-recovery) ────
Sleep 400ms
Type 'usher -p "fix the flaky auth test"'
Sleep 400ms
Enter
Sleep 5800ms

Hide
Type "clear"
Enter
Show

# ── Scene 4: the bow (animated banner) ──────────────────────────
Sleep 400ms
Type "usher"
Sleep 300ms
Enter
Sleep 6500ms
