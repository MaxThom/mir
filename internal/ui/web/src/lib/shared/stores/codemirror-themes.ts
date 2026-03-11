import { EditorView } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';

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

// Serendipity Midnight — editor shell
const serendipityEditorTheme = EditorView.theme(
	{
		'&': { background: '#0d0f18', color: '#b6c1e4' },
		'.cm-content': { caretColor: '#7aa2f7' },
		'.cm-cursor': { borderLeftColor: '#7aa2f7' },
		'.cm-gutters': {
			background: '#0a0c14',
			borderRight: '1px solid #1a1d2e',
			color: '#3b4265'
		},
		'.cm-lineNumbers .cm-gutterElement': { padding: '0 8px' },
		'.cm-activeLineGutter': { background: '#13162440', color: '#5b6494' },
		'.cm-activeLine': { background: '#13162430' },
		'.cm-selectionBackground': { background: '#1e2a4a' },
		'&.cm-focused .cm-selectionBackground': { background: '#243055' },
		'::selection': { background: '#243055' },
		'.cm-matchingBracket': { color: '#c3e88d', outline: '1px solid #c3e88d55' }
	},
	{ dark: true }
);

// Serendipity Midnight — syntax tokens
const serendipityHighlight = HighlightStyle.define([
	// Keywords: soft lavender
	{ tag: tags.keyword, color: '#a78bdb', fontStyle: 'italic' },
	{ tag: tags.controlKeyword, color: '#a78bdb', fontStyle: 'italic' },
	{ tag: tags.operatorKeyword, color: '#a78bdb' },
	{ tag: tags.moduleKeyword, color: '#a78bdb' },

	// Strings: sky blue
	{ tag: tags.string, color: '#7ab0f5' },
	{ tag: tags.special(tags.string), color: '#89c4f4' },
	{ tag: tags.escape, color: '#f5a97f' },

	// Numbers & booleans: soft peach/orange
	{ tag: tags.number, color: '#f5a97f' },
	{ tag: tags.bool, color: '#f5a97f' },
	{ tag: tags.null, color: '#f5a97f' },

	// Functions & methods: cyan
	{ tag: tags.function(tags.variableName), color: '#58d1eb' },
	{ tag: tags.function(tags.propertyName), color: '#58d1eb' },

	// Property names / keys: soft blue-gray
	{ tag: tags.propertyName, color: '#8db4f5' },

	// Types & classes: periwinkle
	{ tag: tags.typeName, color: '#8db4f5' },
	{ tag: tags.className, color: '#8db4f5' },
	{ tag: tags.namespace, color: '#8db4f5' },

	// Variables & names: foreground
	{ tag: tags.name, color: '#b6c1e4' },
	{ tag: tags.variableName, color: '#b6c1e4' },
	{ tag: tags.definition(tags.variableName), color: '#c3e88d' },

	// Operators: light blue
	{ tag: tags.operator, color: '#89c4f4' },
	{ tag: tags.punctuation, color: '#6272a4' },
	{ tag: tags.separator, color: '#6272a4' },
	{ tag: tags.bracket, color: '#7aa2f7' },

	// Comments: dim purple-gray
	{ tag: tags.comment, color: '#454e6b', fontStyle: 'italic' },
	{ tag: tags.lineComment, color: '#454e6b', fontStyle: 'italic' },
	{ tag: tags.blockComment, color: '#454e6b', fontStyle: 'italic' },

	// Tags (HTML/XML): lavender
	{ tag: tags.tagName, color: '#a78bdb' },
	{ tag: tags.attributeName, color: '#58d1eb' },
	{ tag: tags.attributeValue, color: '#7ab0f5' },

	// Misc
	{ tag: tags.meta, color: '#6272a4' },
	{ tag: tags.invalid, color: '#ff5555', textDecoration: 'underline' }
]);

export const midnightTheme = [serendipityEditorTheme, syntaxHighlighting(serendipityHighlight)];
