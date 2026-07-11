#!/usr/bin/env python3
"""Play the usher banner in a terminal.

4-bit ANSI only (theme-friendly), consecutive same-color cells batched into
single SGR runs, cursor hidden during playback and always restored.
Static fallback (final frame, no motion) when: not a TTY, NO_COLOR set,
USHER_NO_BANNER set, or --static passed.
"""
import json, os, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ANSI = {  # semantic role -> 4-bit SGR fg code
    'letter_dim': 90, 'letter_lit': 37, 'letter_hot': 93, 'letter_white': 97,
    'suit': 97, 'eyes': 30, 'bowtie': 91,
    'beam': 93, 'sparkle': 93, 'tagline': 90,
}


def render(fr, color=True):
    lines = fr['content'].split('\n')
    colors = fr.get('colors', {})
    out = []
    for r, line in enumerate(lines):
        run, cur = [], None
        for c, ch in enumerate(line):
            code = ANSI.get(colors.get(f"{r},{c}"), 0) if ch != ' ' else None
            if color and code != cur:
                run.append(f"\x1b[{code}m" if code is not None else "\x1b[0m")
                cur = code
            run.append(ch)
        out.append(''.join(run) + ("\x1b[0m" if color else ""))
    return '\n'.join(out)


def main():
    with open(os.path.join(HERE, 'frames.json')) as f:
        data = json.load(f)
    frames, h = data['frames'], data['height']

    static = ('--static' in sys.argv or not sys.stdout.isatty()
              or os.environ.get('NO_COLOR') or os.environ.get('USHER_NO_BANNER'))
    if static:
        print(render(frames[-1], color=sys.stdout.isatty()))
        return

    sys.stdout.write("\x1b[?25l")  # hide cursor
    try:
        for i, fr in enumerate(frames):
            if i:
                sys.stdout.write(f"\x1b[{h}A")  # cursor back to frame top
            sys.stdout.write(render(fr) + "\n")
            sys.stdout.flush()
            time.sleep(fr['duration'] / 1000)
    finally:
        sys.stdout.write("\x1b[?25h")
        sys.stdout.flush()


if __name__ == '__main__':
    main()
