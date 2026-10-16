import type { ActionId } from "../core/keymap"

export interface MenuDefinition {
  en: string
  pt: string
  actions: ActionId[]
  altKeys: string[]
  mnemonicEn: string
  mnemonicPt: string
}

export const menus: MenuDefinition[] = [
  { en: "File", pt: "Arquivo", actions: ["new_tab", "open_file_fuzzy", "open_folder", "save", "save_as", "save_all", "reload_file", "close_tab", "quit"], altKeys: ["f", "a"], mnemonicEn: "F", mnemonicPt: "A" },
  { en: "Edit", pt: "Editar", actions: ["undo", "redo", "cut", "copy", "paste", "select_all", "find", "replace", "project_find", "project_replace", "toggle_comment", "lsp_rename", "surround", "multi_picker"], altKeys: ["e"], mnemonicEn: "E", mnemonicPt: "E" },
  { en: "View", pt: "Exibir", actions: ["command_palette", "toggle_agents", "toggle_tree", "toggle_terminal", "toggle_scm", "toggle_md_preview", "toggle_soft_wrap", "split_vertical", "split_horizontal", "close_pane", "select_theme", "select_locale", "toggle_mouse", "toggle_modal"], altKeys: ["v", "x"], mnemonicEn: "V", mnemonicPt: "X" },
  { en: "Settings", pt: "Configurações", actions: ["toggle_soft_wrap", "toggle_auto_pairs", "toggle_line_numbers", "toggle_modal", "toggle_icons", "toggle_mouse", "select_theme", "select_locale", "open_user_config", "open_workspace_config", "reload_config"], altKeys: ["s", "c"], mnemonicEn: "S", mnemonicPt: "C" },
  { en: "Go", pt: "Ir", actions: ["goto_line", "buffer_picker", "jump_back", "jump_forward", "lsp_goto_definition", "lsp_hover", "lsp_complete", "lsp_format", "lsp_rename", "toggle_diagnostics", "next_tab", "prev_tab"], altKeys: ["g", "i"], mnemonicEn: "G", mnemonicPt: "I" },
  { en: "Git", pt: "Git", actions: ["toggle_scm", "git_pull", "git_push", "show_diff", "git_blame", "tree_refresh"], altKeys: ["t", "g"], mnemonicEn: "T", mnemonicPt: "T" },
  { en: "Help", pt: "Ajuda", actions: ["help", "which_key", "health_check", "run_tasks", "undo_tree"], altKeys: ["h", "j"], mnemonicEn: "H", mnemonicPt: "J" },
]
