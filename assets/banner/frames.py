#!/usr/bin/env python3
"""Generate usher's animated banner frames.

Architecture follows GitHub Copilot CLI's banner engineering:
plain-text frame content + a separate color layer mapping "row,col" to a
semantic role, colorized at render time with a 4-bit ANSI palette.
Output: frames.json  [{title, duration, content, colors}]
"""
import json, os

W, H = 78, 11          # grid (Copilot's banner is 11x78 too — good omen)
WORD_ROW, WORD_COL = 1, 10   # letters occupy rows 1-8
ARM_ROW = 6                  # flashlight beam row
TAGLINE_ROW, TAGLINE_COL = 10, 10
TAGLINE = "right this way."

# 6x8 pixel glyphs, rendered 2 terminal cells per pixel ("██")
GLYPHS = {
    'u': ["110011","110011","110011","110011","110011","110011","111111","111111"],
    's': ["111111","111111","110000","111111","111111","000011","111111","111111"],
    'h': ["110000","110000","110000","111111","111111","110011","110011","110011"],
    'e': ["111111","111111","110000","111111","111111","110000","111111","111111"],
    'r': ["111111","111111","110011","110011","110000","110000","110000","110000"],
}
WORD = "usher"
LETTER_W = 14  # 6 px * 2 cells + 2 gap

# The usher himself: 6 cols x 6 rows, facing right. Bottom-aligned with letters.
# body rows: head-top, face, torso(+bowtie), arms, legs, feet
DUDE = {
    'stand': [" ▄██▄ ",
              " ████ ",
              " ████ ",
              "▐████▌",
              "  █ █ ",
              "  ▀ ▀ "],
    'walkA': [" ▄██▄ ",
              " ████ ",
              " ████ ",
              "▐████▌",
              " █  █ ",
              " ▀  ▀ "],
    'walkB': [" ▄██▄ ",
              " ████ ",
              " ████ ",
              "▐████▌",
              "  ██  ",
              "  ▀▀  "],
    'point': [" ▄██▄ ",
              " ████ ",
              " ████ ",
              "▐█████",
              "  █ █ ",
              "  ▀ ▀ "],
}
DUDE_TOP = 3  # sprite rows 3..8
# sprite-local (row, col) -> semantic role overrides
EYES = [(1, 2), (1, 4)]
BOWTIE = [(2, 2), (2, 3)]


def blank():
    return [[' '] * W for _ in range(H)]


def paint(grid, colors, r, c, ch, role):
    if 0 <= r < H and 0 <= c < W and ch != ' ':
        grid[r][c] = ch
        colors[f"{r},{c}"] = role


def draw_word(grid, colors, lit_count, hot_index, final=False):
    """lit_count letters already lit; hot_index currently under the beam (-1 none).
    final=True settles into the logo's two-tone: 'ush' gray, 'er' white."""
    for i, ch in enumerate(WORD):
        if final:
            role = 'letter_lit' if i < 3 else 'letter_white'
        elif i == hot_index:
            role = 'letter_hot'
        elif i < lit_count:
            role = 'letter_lit'
        else:
            role = 'letter_dim'
        gx = WORD_COL + i * LETTER_W
        for y, row in enumerate(GLYPHS[ch]):
            for x, bit in enumerate(row):
                if bit == '1':
                    paint(grid, colors, WORD_ROW + y, gx + x * 2, '█', role)
                    paint(grid, colors, WORD_ROW + y, gx + x * 2 + 1, '█', role)


def draw_dude(grid, colors, x, pose, blink=False):
    art = DUDE[pose]
    for ry, row in enumerate(art):
        for rx, ch in enumerate(row):
            if ch == ' ':
                continue
            role = 'suit'
            if (ry, rx) in EYES and not blink:
                role = 'eyes'
            elif (ry, rx) in BOWTIE:
                role = 'bowtie'
            paint(grid, colors, DUDE_TOP + ry, x + rx, ch, role)


def draw_beam(grid, colors, x_from, x_to):
    for c in range(x_from, x_to):
        if grid[ARM_ROW][c] == ' ':  # pass behind letters, never overwrite them
            paint(grid, colors, ARM_ROW, c, '─', 'beam')


def draw_tagline(grid, colors, n_chars):
    for i, ch in enumerate(TAGLINE[:n_chars]):
        if ch != ' ':
            paint(grid, colors, TAGLINE_ROW, TAGLINE_COL + i, ch, 'tagline')


def draw_sparkle(grid, colors, on=True):
    if on:
        paint(grid, colors, 0, WORD_COL + 4 * LETTER_W + 6, '✦', 'sparkle')


def frame(title, duration, painter):
    grid, colors = blank(), {}
    painter(grid, colors)
    return {"title": title, "duration": duration,
            "content": "\n".join("".join(r).rstrip() for r in grid),
            "colors": colors}


def build():
    frames = []
    dude_home = 1

    # 1) walk in from off-screen, letters dim
    walk_xs = [-5, -3, -1, 1]
    for i, x in enumerate(walk_xs):
        pose = 'walkA' if i % 2 == 0 else 'walkB'
        frames.append(frame(f"walk{i}", 100, lambda g, c, x=x, p=pose: (
            draw_word(g, c, 0, -1), draw_dude(g, c, x, p))))

    # 2) stop, raise the flashlight
    frames.append(frame("raise", 120, lambda g, c: (
        draw_word(g, c, 0, -1), draw_dude(g, c, dude_home, 'point'))))

    # 3) sweep: light each letter, two beats each
    arm_tip = dude_home + 6
    for i in range(len(WORD)):
        beam_to = WORD_COL + i * LETTER_W
        for beat in range(2):
            frames.append(frame(f"light-{WORD[i]}{beat}", 85, lambda g, c, i=i, bt=beam_to: (
                draw_word(g, c, i, i), draw_dude(g, c, dude_home, 'point'),
                draw_beam(g, c, arm_tip, bt))))

    # 4) beam off, everything lit, sparkle + tagline types out
    steps = [5, 10, len(TAGLINE)]
    for si, n in enumerate(steps):
        frames.append(frame(f"tagline{si}", 130, lambda g, c, n=n: (
            draw_word(g, c, 5, -1, final=True), draw_dude(g, c, dude_home, 'stand'),
            draw_sparkle(g, c), draw_tagline(g, c, n))))

    # 5) a blink for good measure, then hold
    frames.append(frame("blink", 120, lambda g, c: (
        draw_word(g, c, 5, -1, final=True), draw_dude(g, c, dude_home, 'stand', blink=True),
        draw_sparkle(g, c), draw_tagline(g, c, len(TAGLINE)))))
    frames.append(frame("hold", 600, lambda g, c: (
        draw_word(g, c, 5, -1, final=True), draw_dude(g, c, dude_home, 'stand'),
        draw_sparkle(g, c), draw_tagline(g, c, len(TAGLINE)))))
    return frames


if __name__ == '__main__':
    out = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'frames.json')
    frames = build()
    with open(out, 'w') as f:
        json.dump({"width": W, "height": H, "frames": frames}, f)
    total = sum(fr['duration'] for fr in frames)
    print(f"{len(frames)} frames, {total}ms total → {out}")
