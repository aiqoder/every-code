import { Node, mergeAttributes } from '@tiptap/core'

export type SkillChipAttrs = {
  id: string
  name: string
}

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    skillChip: {
      insertSkillChip: (attrs: SkillChipAttrs) => ReturnType
    }
  }
}

export const SkillChip = Node.create({
  name: 'skillChip',
  group: 'inline',
  inline: true,
  atom: true,
  selectable: true,
  draggable: false,

  addAttributes() {
    return {
      id: { default: '' },
      name: { default: '' },
    }
  },

  parseHTML() {
    return [
      {
        tag: 'span[data-skill-id]',
        getAttrs: (el) => {
          if (!(el instanceof HTMLElement)) return false
          return {
            id: el.getAttribute('data-skill-id') || '',
            name: el.getAttribute('data-skill-name') || el.textContent || '',
          }
        },
      },
    ]
  },

  renderHTML({ node, HTMLAttributes }) {
    const name = String(node.attrs.name || node.attrs.id || '')
    return [
      'span',
      mergeAttributes(HTMLAttributes, {
        'data-skill-id': node.attrs.id,
        'data-skill-name': name,
        class: 'skill-chip',
        contenteditable: 'false',
      }),
      `/${name}`,
    ]
  },

  addCommands() {
    return {
      insertSkillChip:
        (attrs) =>
        ({ commands }) =>
          commands.insertContent({
            type: this.name,
            attrs: {
              id: attrs.id,
              name: attrs.name,
            },
          }),
    }
  },
})
