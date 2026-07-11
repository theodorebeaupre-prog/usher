#!/usr/bin/env python3
"""Generate pixel wordmark for 'usher' — SVG + PNG, two styles."""
import struct, zlib, os

# 8-row grid, 6-wide glyphs, 2px strokes
GLYPHS = {
    'u': ["110011","110011","110011","110011","110011","110011","111111","111111"],
    's': ["111111","111111","110000","111111","111111","000011","111111","111111"],
    'h': ["110000","110000","110000","111111","111111","110011","110011","110011"],
    'e': ["111111","111111","110000","111111","111111","110000","111111","111111"],
    'r': ["111111","111111","110011","110011","110000","110000","110000","110000"],
}

def word_grid(word, gap=1, pad=2):
    rows = 8
    cells = {}  # (x,y) -> letter index
    x0 = pad
    for i, ch in enumerate(word):
        g = GLYPHS[ch]
        for y in range(rows):
            for x in range(6):
                if g[y][x] == '1':
                    cells[(x0 + x, pad + y)] = i
        x0 += 6 + gap
    w = x0 - gap + pad
    h = rows + 2 * pad
    return cells, w, h

def hex2rgb(s):
    return tuple(int(s[i:i+2], 16) for i in (1, 3, 5))

def render(word, style, scale=16):
    cells, W, H = word_grid(word)
    px = {}  # (x,y) -> color hex

    if style == 'gray':
        bg = '#0d0d0d'
        for (x, y), i in cells.items():
            base = '#6e6e6e' if i < 3 else '#ececec'
            hi   = '#8f8f8f' if i < 3 else '#ffffff'
            lo   = '#4f4f4f' if i < 3 else '#b9b9b9'
            top_open = (x, y - 1) not in cells
            bot_open = (x, y + 1) not in cells
            px[(x, y)] = hi if top_open else (lo if bot_open else base)
    else:  # orange, Claude-Code-ish with drop shadow
        bg = '#101010'
        sh = '#4a2318'
        for (x, y) in cells:
            px.setdefault((x + 1, y + 1), sh)  # shadow first (under)
        for (x, y), i in cells.items():
            top_open = (x, y - 1) not in cells
            bot_open = (x, y + 1) not in cells
            px[(x, y)] = '#f0906c' if top_open else ('#b85a3c' if bot_open else '#d97757')

    # SVG
    svg = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="{W*scale}" height="{H*scale}" shape-rendering="crispEdges">']
    svg.append(f'<rect width="{W}" height="{H}" fill="{bg}"/>')
    # shadow rects first, then letters (dict preserves insertion but letters may
    # have been set after shadow via setdefault ordering; just sort: shadow color first)
    order = sorted(px.items(), key=lambda kv: 0 if kv[1] == '#4a2318' else 1)
    for (x, y), c in order:
        svg.append(f'<rect x="{x}" y="{y}" width="1" height="1" fill="{c}"/>')
    svg.append('</svg>')
    svg_text = '\n'.join(svg)

    # PNG (pure python)
    bgc = hex2rgb(bg)
    img = [[bgc] * (W * scale) for _ in range(H * scale)]
    for (x, y), c in order:
        rgb = hex2rgb(c)
        for yy in range(y * scale, (y + 1) * scale):
            row = img[yy]
            for xx in range(x * scale, (x + 1) * scale):
                row[xx] = rgb
    raw = b''.join(b'\x00' + b''.join(struct.pack('BBB', *p) for p in row) for row in img)
    def chunk(t, d):
        c = t + d
        return struct.pack('>I', len(d)) + c + struct.pack('>I', zlib.crc32(c))
    png = (b'\x89PNG\r\n\x1a\n'
           + chunk(b'IHDR', struct.pack('>IIBBBBB', W * scale, H * scale, 8, 2, 0, 0, 0))
           + chunk(b'IDAT', zlib.compress(raw, 9))
           + chunk(b'IEND', b''))
    return svg_text, png

out = os.environ.get('OUT', '.')
for style in ('gray', 'orange'):
    svg_text, png = render('usher', style)
    with open(f'{out}/usher-{style}.svg', 'w') as f: f.write(svg_text)
    with open(f'{out}/usher-{style}.png', 'wb') as f: f.write(png)
    print(f'wrote usher-{style}.svg / .png')
