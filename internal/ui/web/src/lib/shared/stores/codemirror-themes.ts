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

// Hacker — editor shell
const hackerEditorTheme = EditorView.theme(
	{
		'&': { background: '#020c02', color: '#00ff41' },
		'.cm-content': { caretColor: '#00ff41' },
		'.cm-cursor': { borderLeftColor: '#00ff41' },
		'.cm-gutters': {
			background: '#010801',
			borderRight: '1px solid #003810',
			color: '#1a5c1a'
		},
		'.cm-lineNumbers .cm-gutterElement': { padding: '0 8px' },
		'.cm-activeLineGutter': { background: '#001a0040', color: '#2a8a2a' },
		'.cm-activeLine': { background: '#00ff4108' },
		'.cm-selectionBackground': { background: '#00802b40' },
		'&.cm-focused .cm-selectionBackground': { background: '#00802b55' },
		'::selection': { background: '#00802b55' },
		'.cm-matchingBracket': { color: '#69ff47', outline: '1px solid #69ff4755' }
	},
	{ dark: true }
);

// Hacker — syntax tokens in shades of green
const hackerHighlight = HighlightStyle.define([
	{ tag: tags.keyword, color: '#69ff47', fontStyle: 'italic' },
	{ tag: tags.controlKeyword, color: '#69ff47', fontStyle: 'italic' },
	{ tag: tags.operatorKeyword, color: '#69ff47' },
	{ tag: tags.moduleKeyword, color: '#69ff47' },

	{ tag: tags.string, color: '#00c853' },
	{ tag: tags.special(tags.string), color: '#00e676' },
	{ tag: tags.escape, color: '#b9ffce' },

	{ tag: tags.number, color: '#a3ffba' },
	{ tag: tags.bool, color: '#a3ffba' },
	{ tag: tags.null, color: '#a3ffba' },

	{ tag: tags.function(tags.variableName), color: '#39ff14' },
	{ tag: tags.function(tags.propertyName), color: '#39ff14' },

	{ tag: tags.propertyName, color: '#00e676' },

	{ tag: tags.typeName, color: '#69ff47' },
	{ tag: tags.className, color: '#69ff47' },
	{ tag: tags.namespace, color: '#69ff47' },

	{ tag: tags.name, color: '#00ff41' },
	{ tag: tags.variableName, color: '#00ff41' },
	{ tag: tags.definition(tags.variableName), color: '#b9ffce' },

	{ tag: tags.operator, color: '#00802b' },
	{ tag: tags.punctuation, color: '#006622' },
	{ tag: tags.separator, color: '#006622' },
	{ tag: tags.bracket, color: '#00802b' },

	{ tag: tags.comment, color: '#1a4a1a', fontStyle: 'italic' },
	{ tag: tags.lineComment, color: '#1a4a1a', fontStyle: 'italic' },
	{ tag: tags.blockComment, color: '#1a4a1a', fontStyle: 'italic' },

	{ tag: tags.tagName, color: '#69ff47' },
	{ tag: tags.attributeName, color: '#39ff14' },
	{ tag: tags.attributeValue, color: '#00c853' },

	{ tag: tags.meta, color: '#006622' },
	{ tag: tags.invalid, color: '#ff0000', textDecoration: 'underline' }
]);

export const hackerTheme = [hackerEditorTheme, syntaxHighlighting(hackerHighlight)];

// Catppuccin Mocha — editor shell
const mochaEditorTheme = EditorView.theme(
	{
		'&': { background: '#1e1e2e', color: '#cdd6f4' },
		'.cm-content': { caretColor: '#f5e0dc' },
		'.cm-cursor': { borderLeftColor: '#f5e0dc' },
		'.cm-gutters': {
			background: '#181825',
			borderRight: '1px solid #313244',
			color: '#6c7086'
		},
		'.cm-lineNumbers .cm-gutterElement': { padding: '0 8px' },
		'.cm-activeLineGutter': { background: '#31324480', color: '#9399b2' },
		'.cm-activeLine': { background: '#31324430' },
		'.cm-selectionBackground': { background: '#45475a' },
		'&.cm-focused .cm-selectionBackground': { background: '#585b70' },
		'::selection': { background: '#585b70' },
		'.cm-matchingBracket': { color: '#a6e3a1', outline: '1px solid #a6e3a155' }
	},
	{ dark: true }
);

// Catppuccin Mocha — syntax tokens
const mochaHighlight = HighlightStyle.define([
	{ tag: tags.keyword, color: '#cba6f7', fontStyle: 'italic' },
	{ tag: tags.controlKeyword, color: '#cba6f7', fontStyle: 'italic' },
	{ tag: tags.operatorKeyword, color: '#cba6f7' },
	{ tag: tags.moduleKeyword, color: '#cba6f7' },

	{ tag: tags.string, color: '#a6e3a1' },
	{ tag: tags.special(tags.string), color: '#94e2d5' },
	{ tag: tags.escape, color: '#f2cdcd' },

	{ tag: tags.number, color: '#fab387' },
	{ tag: tags.bool, color: '#fab387' },
	{ tag: tags.null, color: '#f38ba8' },

	{ tag: tags.function(tags.variableName), color: '#89b4fa' },
	{ tag: tags.function(tags.propertyName), color: '#89b4fa' },

	{ tag: tags.propertyName, color: '#89dceb' },

	{ tag: tags.typeName, color: '#f9e2af' },
	{ tag: tags.className, color: '#f9e2af' },
	{ tag: tags.namespace, color: '#f9e2af' },

	{ tag: tags.name, color: '#cdd6f4' },
	{ tag: tags.variableName, color: '#cdd6f4' },
	{ tag: tags.definition(tags.variableName), color: '#89b4fa' },

	{ tag: tags.operator, color: '#89dceb' },
	{ tag: tags.punctuation, color: '#9399b2' },
	{ tag: tags.separator, color: '#9399b2' },
	{ tag: tags.bracket, color: '#b4befe' },

	{ tag: tags.comment, color: '#6c7086', fontStyle: 'italic' },
	{ tag: tags.lineComment, color: '#6c7086', fontStyle: 'italic' },
	{ tag: tags.blockComment, color: '#6c7086', fontStyle: 'italic' },

	{ tag: tags.tagName, color: '#cba6f7' },
	{ tag: tags.attributeName, color: '#89b4fa' },
	{ tag: tags.attributeValue, color: '#a6e3a1' },

	{ tag: tags.meta, color: '#7f849c' },
	{ tag: tags.invalid, color: '#f38ba8', textDecoration: 'underline' }
]);

export const mochaTheme = [mochaEditorTheme, syntaxHighlighting(mochaHighlight)];
