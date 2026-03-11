import { EditorView } from '@codemirror/view';

export const rustTheme = EditorView.theme(
	{
		'&': { background: '#f4ece2', color: '#2c2118' },
		'.cm-content': { caretColor: '#8b5e3c' },
		'.cm-gutters': {
			background: '#ede4d6',
			borderRight: '1px solid #c9b49a',
			color: '#8b7355'
		},
		'.cm-activeLineGutter': { background: '#e4d8c8' },
		'.cm-activeLine': { background: '#ede4d640' },
		'.cm-selectionBackground, ::selection': { background: '#c9b49a55' }
	},
	{ dark: false }
);
