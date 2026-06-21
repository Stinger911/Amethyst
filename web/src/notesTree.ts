import type { NoteSummary } from './api'

export interface FolderNode {
  type: 'folder'
  name: string
  path: string
  children: TreeNode[]
}

export interface NoteNode {
  type: 'note'
  name: string
  path: string
  title: string
  tags: string[]
}

export type TreeNode = FolderNode | NoteNode

// Notes carry a full vault-relative path (e.g. "Folder/Sub/Note.md") with no
// separate directory listing from the API — the tree is reconstructed
// client-side by splitting on '/'.
export function buildNotesTree(notes: NoteSummary[]): TreeNode[] {
  const root: TreeNode[] = []

  for (const note of notes) {
    const parts = note.path.split('/')
    let siblings = root
    let prefix = ''

    for (let i = 0; i < parts.length - 1; i++) {
      const name = parts[i]
      prefix = prefix ? `${prefix}/${name}` : name
      let folder = siblings.find(
        (n): n is FolderNode => n.type === 'folder' && n.name === name,
      )
      if (!folder) {
        folder = { type: 'folder', name, path: prefix, children: [] }
        siblings.push(folder)
      }
      siblings = folder.children
    }

    siblings.push({
      type: 'note',
      name: parts[parts.length - 1],
      path: note.path,
      title: note.title,
      tags: note.tags,
    })
  }

  sortTree(root)
  return root
}

function sortTree(nodes: TreeNode[]) {
  nodes.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'folder' ? -1 : 1
    return a.name.localeCompare(b.name)
  })
  for (const node of nodes) {
    if (node.type === 'folder') sortTree(node.children)
  }
}
