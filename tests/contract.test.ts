import { expect, test } from "bun:test"
import { actionIds } from "../src/core/keymap"
import { commands } from "../src/app/commands"
import { menus } from "../src/app/menus"

test("action IDs and command registry are exhaustive mirrors", () => {
  expect(Object.keys(commands).sort()).toEqual([...actionIds].sort())
})

test("menu actions refer to existing command IDs", () => {
  for (const menu of menus) for (const action of menu.actions) expect(actionIds.includes(action)).toBe(true)
})

test("command labels are non-empty and unique", () => {
  const labels = Object.values(commands).map(command => command.label)
  expect(labels.every(label => label.trim().length > 0)).toBe(true)
  expect(new Set(labels).size).toBe(labels.length)
})
