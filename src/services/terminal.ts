import xterm from "@xterm/headless"
import type { IBufferCell } from "@xterm/headless"

export type TerminalCell = { text: string; width: number; foreground?: number; background?: number; foregroundPalette: boolean; backgroundPalette: boolean; bold: boolean; italic: boolean; underline: boolean; inverse: boolean }

export class TerminalSession {
  private emulator: xterm.Terminal
  private terminal: Bun.Terminal
  private process: Bun.Subprocess
  private closed = false
  exitCode: number | undefined
  scroll = 0
  cols: number
  rows: number

  constructor(shell: string, cwd: string, cols: number, rows: number, readonly onChange: () => void) {
    this.cols = cols; this.rows = rows
    this.emulator = new xterm.Terminal({ cols, rows, scrollback: 2000, allowProposedApi: true })
    this.terminal = new Bun.Terminal({ cols, rows, data: (_terminal, bytes) => {
      if (!this.closed) this.emulator.write(bytes, () => { if (!this.closed) this.onChange() })
    } })
    try {
      this.process = Bun.spawn([shell, "-i"], { cwd, terminal: this.terminal, detached: true, env: { ...process.env, TERM: "xterm-256color" } })
    } catch (error) {
      this.terminal.close(); this.emulator.dispose(); throw error
    }
    this.emulator.onData(text => { if (!this.closed) this.terminal.write(text) })
    void this.process.exited.then(code => { this.exitCode = code; if (!this.closed) this.onChange() })
  }

  write(text: string): void {
    if (this.closed || this.exitCode !== undefined) throw new Error("Terminal is closed")
    this.scroll = 0
    this.terminal.write(text)
  }

  resize(cols: number, rows: number): void {
    cols = Math.max(1, cols); rows = Math.max(1, rows)
    if (cols === this.cols && rows === this.rows) return
    this.cols = cols; this.rows = rows
    this.terminal.resize(cols, rows)
    this.emulator.resize(cols, rows)
  }

  lines(): string[] {
    const buffer = this.emulator.buffer.active
    const start = Math.max(0, buffer.baseY - this.scroll)
    return Array.from({ length: this.rows }, (_, row) => buffer.getLine(start + row)?.translateToString(true) ?? "")
  }

  cells(): TerminalCell[][] {
    const buffer = this.emulator.buffer.active
    const start = Math.max(0, buffer.baseY - this.scroll)
    const convert = (cell: IBufferCell): TerminalCell => ({
      text: cell.getChars() || " ", width: cell.getWidth(),
      foreground: cell.isFgDefault() ? undefined : cell.getFgColor(), background: cell.isBgDefault() ? undefined : cell.getBgColor(),
      foregroundPalette: !!cell.isFgPalette(), backgroundPalette: !!cell.isBgPalette(),
      bold: !!cell.isBold(), italic: !!cell.isItalic(), underline: !!cell.isUnderline(), inverse: !!cell.isInverse(),
    })
    return Array.from({ length: this.rows }, (_, row) => {
      const line = buffer.getLine(start + row)
      return Array.from({ length: this.cols }, (_, column) => line?.getCell(column)).filter((cell): cell is IBufferCell => !!cell).map(convert)
    })
  }

  get cursor(): { row: number; column: number } {
    return { row: this.emulator.buffer.active.cursorY, column: this.emulator.buffer.active.cursorX }
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    if (this.process.exitCode === null) {
      try { process.kill(-this.process.pid, "SIGKILL") } catch (error) {
        if (!(error instanceof Error && "code" in error && error.code === "ESRCH")) throw error
      }
    }
    this.terminal.close()
    this.process.kill("SIGKILL")
    await this.process.exited
    this.emulator.dispose()
  }
}
