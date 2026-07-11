#!/usr/bin/env python3
"""Render frames.json to PNG frames (pure python) and a GIF (via ffmpeg)."""
import json, os, struct, subprocess, sys, zlib

HERE = os.path.dirname(os.path.abspath(__file__))
CW, CH = 8, 16  # pixels per terminal cell

BG = (0x0d, 0x0d, 0x0d)
RGB = {  # semantic role -> rgb, mirroring a dark-terminal 4-bit palette
    'letter_dim': (0x55, 0x55, 0x55), 'letter_lit': (0xc9, 0xc9, 0xc9),
    'letter_hot': (0xff, 0xd7, 0x5f), 'letter_white': (0xf5, 0xf5, 0xf5),
    'suit': (0xf2, 0xf2, 0xf2),
    'eyes': BG, 'bowtie': (0xff, 0x5c, 0x5c), 'beam': (0xff, 0xd7, 0x5f),
    'sparkle': (0xff, 0xd7, 0x5f), 'tagline': (0x8a, 0x8a, 0x8a),
}

FONT = {  # 3x5 minifont for the tagline (w is 5 wide)
    'r': ["000","000","111","100","100"], 'i': ["010","000","010","010","010"],
    'g': ["111","101","111","001","111"], 'h': ["100","100","111","101","101"],
    't': ["010","111","010","010","011"], 's': ["111","100","111","001","111"],
    'w': ["00000","00000","10101","10101","01110"], 'a': ["000","011","101","101","011"],
    'y': ["101","101","111","001","110"], '.': ["000","000","000","000","100"],
}


def cell_pixels(ch):
    """Which subpixels of a CW x CH cell are on for this char."""
    on = set()
    if ch == '█':
        on = {(x, y) for x in range(CW) for y in range(CH)}
    elif ch == '▀':
        on = {(x, y) for x in range(CW) for y in range(CH // 2)}
    elif ch == '▄':
        on = {(x, y) for x in range(CW) for y in range(CH // 2, CH)}
    elif ch == '▌':
        on = {(x, y) for x in range(CW // 2) for y in range(CH)}
    elif ch == '▐':
        on = {(x, y) for x in range(CW // 2, CW) for y in range(CH)}
    elif ch == '─':
        on = {(x, y) for x in range(CW) for y in range(7, 9)}
    elif ch == '✦':
        cx, cy, r = CW // 2, CH // 2, 5
        on = {(x, y) for x in range(CW) for y in range(CH)
              if abs(x - cx) * 2 + abs(y - cy) <= r}
    elif ch in FONT:
        g = FONT[ch]
        for gy, row in enumerate(g):
            for gx, bit in enumerate(row):
                if bit == '1':
                    for dx in range(2):
                        for dy in range(2):
                            on.add((gx * 2 + dx + 1, gy * 2 + dy + 3))
    return on


def png_write(path, img, w, h):
    raw = b''.join(b'\x00' + b''.join(struct.pack('BBB', *p) for p in row) for row in img)
    def chunk(t, d):
        c = t + d
        return struct.pack('>I', len(d)) + c + struct.pack('>I', zlib.crc32(c))
    with open(path, 'wb') as f:
        f.write(b'\x89PNG\r\n\x1a\n'
                + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
                + chunk(b'IDAT', zlib.compress(raw, 9))
                + chunk(b'IEND', b''))


def render_frame(fr, W, H):
    img = [[BG] * (W * CW) for _ in range(H * CH)]
    colors = fr.get('colors', {})
    for r, line in enumerate(fr['content'].split('\n')):
        for c, ch in enumerate(line):
            if ch == ' ':
                continue
            rgb = RGB.get(colors.get(f"{r},{c}"), (0xc9,) * 3)
            for (x, y) in cell_pixels(ch):
                img[r * CH + y][c * CW + x] = rgb
    return img


def main():
    with open(os.path.join(HERE, 'frames.json')) as f:
        data = json.load(f)
    W, H = data['width'], data['height']
    outdir = os.path.join(HERE, 'png-frames')
    os.makedirs(outdir, exist_ok=True)
    concat = []
    last = ''
    for i, fr in enumerate(data['frames']):
        last = os.path.join(outdir, f"f{i:03d}.png")
        png_write(last, render_frame(fr, W, H), W * CW, H * CH)
        concat.append(f"file '{last}'\nduration {fr['duration'] / 1000}")
    concat.append(f"file '{last}'")  # concat demuxer needs the final file repeated
    print(f"wrote {len(data['frames'])} pngs")

    listfile = os.path.join(outdir, 'list.txt')
    with open(listfile, 'w') as f:
        f.write('\n'.join(concat))
    gif = os.path.join(HERE, 'usher-banner.gif')
    subprocess.run(['ffmpeg', '-y', '-loglevel', 'error', '-f', 'concat', '-safe', '0',
                    '-i', listfile, '-vf',
                    'split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse',
                    '-loop', '0', gif], check=True)
    print(f"wrote {gif}")


if __name__ == '__main__':
    main()
