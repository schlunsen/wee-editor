const ESC = '\x1b[';
const ink = `${ESC}38;2;225;216;249m`;
const muted = `${ESC}38;2;130;125;146m`;
const accent = `${ESC}38;2;180;151;243m`;
const reset = `${ESC}0m`;
const logo = [
  '██╗    ██╗███████╗███████╗',
  '██║    ██║██╔════╝██╔════╝',
  '██║ █╗ ██║█████╗  █████╗  ',
  '██║███╗██║██╔══╝  ██╔══╝  ',
  '╚███╔███╔╝███████╗███████╗',
  ' ╚══╝╚══╝ ╚══════╝╚══════╝',
];

export function splashOverlay({ columns = 80, rows = 24, elapsed = 0, stage = 0, total = 2 } = {}) {
  if (rows < 14 || columns < 32) return '';
  const center = (text, row, color = ink) => {
    const clipped = text.slice(0, Math.max(1, columns - 4));
    const col = Math.max(1, Math.floor((columns - clipped.length) / 2) + 1);
    return `${ESC}${row};1H${ESC}48;2;0;0;0m${ESC}2K${ESC}${row};${col}H${color}${clipped}${reset}`;
  };
  let out = '\x1b7';
  const spacious = rows >= 32;
  if (spacious) {
    const reveal = Math.min(logo[0].length, Math.floor(elapsed / 24) + 1);
    logo.forEach((line, index) => { out += center(line.slice(0, reveal), index + 2, accent); });
  } else out += center('w e e', 2, accent);
  const tagline = 'Your agents. One place.';
  const typed = tagline.slice(0, Math.max(0, Math.floor((elapsed - 500) / 45)));
  out += center(typed + (elapsed < 1900 ? '▏' : ''), spacious ? 9 : 4, muted);

  const seconds = Math.floor(elapsed / 1000);
  const clock = `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
  const stageName = stage === 0 ? 'Building the editor' : stage === 1 ? 'Compiling the engine' : 'Finishing the build';
  const dots = '.'.repeat(Math.floor(elapsed / 360) % 4).padEnd(3, ' ');
  out += center(`${stageName}${dots}`, rows - 6);
  const width = Math.min(30, columns - 12), cycle = width * 2 - 8;
  const tick = Math.floor(elapsed / 90) % cycle;
  const position = tick < width - 4 ? tick : cycle - tick;
  const track = Array.from({ length: width }, (_, i) => i >= position && i < position + 4 ? '━' : '─').join('');
  out += center(track, rows - 4, accent);
  out += center(`${String(stage + 1).padStart(2, '0')} / ${String(total).padStart(2, '0')}   ${stage > 0 ? '✓ Frontend ready   ·   ' : ''}${clock}`, rows - 2, muted);
  out += center('Q  Hide animation     Ctrl+C  Cancel build', rows, muted);
  return out + '\x1b8';
}

// Append text inside Rasterminal's synchronized-output frame, before it becomes
// visible. Buffer split frames so partial terminal writes cannot tear the UI.
export function frameCompositor(overlay, write) {
  let pending = '';
  const end = '\x1b[?2026l';
  return {
    feed(text) {
      pending += text;
      let at;
      while ((at = pending.indexOf(end)) >= 0) {
        const frame = pending.slice(0, at);
        write(frame + (frame.includes('\x1b[?2026h') ? overlay() : '') + end);
        pending = pending.slice(at + end.length);
      }
    },
    flush() { if (pending) write(pending); pending = ''; }
  };
}
