import { expect, test } from "bun:test"
import { testRender } from "@opentui/solid"

test("OpenTUI mounts Solid with the native renderer", async () => {
  const setup = await testRender(() => <text>Daisy · OpenTUI/Solid</text>, { width: 40, height: 5 })
  try {
    await setup.renderOnce()
    expect(setup.captureCharFrame()).toContain("Daisy · OpenTUI/Solid")
  } finally {
    setup.renderer.destroy()
  }
})
