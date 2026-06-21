import { describe, expect, it } from 'vitest'
import { buildNotesTree, type FolderNode } from './notesTree'

describe('buildNotesTree', () => {
  it('returns an empty tree for no notes', () => {
    expect(buildNotesTree([])).toEqual([])
  })

  it('places top-level notes directly under the root, sorted by filename', () => {
    const tree = buildNotesTree([
      { path: 'Zeta.md', title: 'Zeta', tags: [] },
      { path: 'Alpha.md', title: 'Alpha', tags: ['x'] },
    ])

    expect(tree).toEqual([
      { type: 'note', name: 'Alpha.md', path: 'Alpha.md', title: 'Alpha', tags: ['x'] },
      { type: 'note', name: 'Zeta.md', path: 'Zeta.md', title: 'Zeta', tags: [] },
    ])
  })

  it('groups notes under nested folders derived from their path', () => {
    const tree = buildNotesTree([
      { path: 'Hub.md', title: 'Hub', tags: [] },
      { path: 'Folder/Leaf.md', title: 'Leaf', tags: ['a'] },
      { path: 'Folder/Sub/Deep.md', title: 'Deep', tags: [] },
    ])

    expect(tree).toHaveLength(2)
    const [folder, hub] = tree
    expect(folder.type).toBe('folder')
    expect(hub).toEqual({ type: 'note', name: 'Hub.md', path: 'Hub.md', title: 'Hub', tags: [] })

    const folderNode = folder as FolderNode
    expect(folderNode.name).toBe('Folder')
    expect(folderNode.path).toBe('Folder')
    expect(folderNode.children).toHaveLength(2)

    const [sub, leaf] = folderNode.children
    expect(leaf).toEqual({
      type: 'note',
      name: 'Leaf.md',
      path: 'Folder/Leaf.md',
      title: 'Leaf',
      tags: ['a'],
    })
    expect(sub.type).toBe('folder')
    expect((sub as FolderNode).path).toBe('Folder/Sub')
    expect((sub as FolderNode).children).toEqual([
      { type: 'note', name: 'Deep.md', path: 'Folder/Sub/Deep.md', title: 'Deep', tags: [] },
    ])
  })

  it('sorts folders before notes at every level', () => {
    const tree = buildNotesTree([
      { path: 'Zeta.md', title: 'Zeta', tags: [] },
      { path: 'Aardvark/Note.md', title: 'Note', tags: [] },
    ])

    expect(tree.map((n) => n.type)).toEqual(['folder', 'note'])
  })

  it('reuses the same folder node for multiple notes sharing a path prefix', () => {
    const tree = buildNotesTree([
      { path: 'Folder/A.md', title: 'A', tags: [] },
      { path: 'Folder/B.md', title: 'B', tags: [] },
    ])

    expect(tree).toHaveLength(1)
    expect((tree[0] as FolderNode).children).toHaveLength(2)
  })
})
