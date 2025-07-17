/*
 * Bamboo - A Vietnamese Input method editor
 * Copyright (C) 2018 Luong Thanh Lam <ltlam93@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/BambooEngine/bamboo-core"
	ibus "github.com/BambooEngine/goibus"
	"github.com/godbus/dbus/v5"

	"ibus-bamboo/config"
)

type IBusBambooEngine struct {
	sync.Mutex
	IEngine
	preeditor              bamboo.IEngine
	engineName             string
	config                 *config.Config
	propList               *ibus.PropList
	englishMode            bool
	macroTable             *MacroTable
	wmClasses              string
	isInputModeLTOpened    bool
	isEmojiLTOpened        bool
	isInHexadecimal        bool
	emojiLookupTable       *ibus.LookupTable
	inputModeLookupTable   *ibus.LookupTable
	capabilities           uint32
	keyPressDelay          int
	nFakeBackSpace         int32
	isFirstTimeSendingBS   bool
	emoji                  *EmojiEngine
	isSurroundingTextReady bool
	lastKeyWithShift       bool
	lastCommitText         int64
	// restore key strokes by pressing Shift + Space
	shouldRestoreKeyStrokes bool
	// enqueue key strokes to process later
	shouldEnqueuKeyStrokes bool
}

func NewIbusBambooEngine(name string, cfg *config.Config, base IEngine, preeditor bamboo.IEngine) *IBusBambooEngine {
	return &IBusBambooEngine{
		engineName: name,
		IEngine:    base,
		preeditor:  preeditor,
		config:     cfg,
	}
}

/*
*
Implement IBus.Engine's process_key_event default signal handler.

Args:

	keyval - The keycode, transformed through a keymap, stays the
		same for every keyboard
	keycode - Keyboard-dependant key code
	modifiers - The state of IBus.ModifierType keys like
		Shift, Control, etc.

Return:

	True - if successfully process the keyevent
	False - otherwise. The keyevent will be passed to X-Client

This function gets called whenever a key is pressed.
*/
func (e *IBusBambooEngine) ProcessKeyEvent(keyVal uint32, keyCode uint32, state uint32) (bool, *dbus.Error) {
	if state&IBusReleaseMask != 0 {
		// fmt.Println("Ignore key-up event")
		return false, nil
	}
	fmt.Printf("\n")
	log.Printf(">>>>ProcessKeyEvent >  %d | state %d keyVal 0x%04x | %c <<<<\n", len(keyPressChan), state, keyVal, rune(keyVal))
	if ret, retValue := e.processShortcutKey(keyVal, keyCode, state); ret {
		return retValue, nil
	}
	if e.inBackspaceWhiteList() {
		return e.bsProcessKeyEvent(keyVal, keyCode, state)
	}
	return e.preeditProcessKeyEvent(keyVal, keyCode, state)
}

func (e *IBusBambooEngine) FocusIn() *dbus.Error {
	log.Print("FocusIn.")
	var latestWm = e.getLatestWmClass()
	e.checkWmClass(latestWm)
	e.RegisterProperties(e.propList)
	e.RequireSurroundingText()
	if e.isShortcutKeyEnable(KSEmojiDialog) && emojiTrie != nil && len(emojiTrie.Children) == 0 {
		var err error
		emojiTrie, err = loadEmojiOne(DictEmojiOne)
		if err != nil {
			panic(fmt.Sprintf("failed to load emojiTrie from %s: %s", DictEmojiOne, err))
		}
	}
	if e.config.IBflags&config.IBspellCheckWithDicts != 0 && len(dictionary) == 0 {
		dictionary, _ = loadDictionary(DictVietnameseCm)
	}
	if inStringList(disabledMouseCapturingList, e.getWmClass()) {
		stopMouseCapturing()
	} else if e.config.IBflags&config.IBmouseCapturing != 0 {
		startMouseCapturing()
	}
	fmt.Printf("WM_CLASS=(%s)\n", e.getWmClass())
	return nil
}

func (e *IBusBambooEngine) FocusOut() *dbus.Error {
	log.Print("FocusOut.")
	return nil
}

func (e *IBusBambooEngine) Reset() *dbus.Error {
	fmt.Print("Reset.\n")
	if e.checkInputMode(config.PreeditIM) {
		e.commitPreeditAndReset(e.getPreeditString())
	}
	return nil
}

func (e *IBusBambooEngine) Enable() *dbus.Error {
	fmt.Print("Enable.")
	e.RequireSurroundingText()
	return nil
}

func (e *IBusBambooEngine) Disable() *dbus.Error {
	fmt.Print("Disable.")
	return nil
}

// @method(in_signature="vuu")
func (e *IBusBambooEngine) SetSurroundingText(text dbus.Variant, cursorPos uint32, anchorPos uint32) *dbus.Error {
	if !e.isSurroundingTextReady {
		//fmt.Println("Surrounding Text is not ready yet.")
		return nil
	}
	e.Lock()
	defer func() {
		e.Unlock()
		e.isSurroundingTextReady = false
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	if e.inBackspaceWhiteList() {
		var str = reflect.ValueOf(reflect.ValueOf(text.Value()).Index(2).Interface()).String()
		var s = []rune(str)
		if len(s) < int(cursorPos) {
			return nil
		}
		var cs = s[:cursorPos]
		fmt.Println("Surrounding Text: ", string(cs))
		e.preeditor.Reset()
		for i := len(cs) - 1; i >= 0; i-- {
			// workaround for spell checking
			if bamboo.IsPunctuationMark(cs[i]) && e.preeditor.CanProcessKey(cs[i]) {
				cs[i] = ' '
			}
			e.preeditor.ProcessKey(cs[i], bamboo.EnglishMode|bamboo.InReverseOrder)
		}
	}
	return nil
}

func (e *IBusBambooEngine) PageUp() *dbus.Error {
	if e.isEmojiLTOpened && e.emojiLookupTable.PageUp() {
		e.updateEmojiLookupTable()
	}
	if e.isInputModeLTOpened && e.inputModeLookupTable.PageUp() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusBambooEngine) PageDown() *dbus.Error {
	if e.isEmojiLTOpened && e.emojiLookupTable.PageDown() {
		e.updateEmojiLookupTable()
	}
	if e.isInputModeLTOpened && e.inputModeLookupTable.PageDown() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusBambooEngine) CursorUp() *dbus.Error {
	if e.isEmojiLTOpened && e.emojiLookupTable.CursorUp() {
		e.updateEmojiLookupTable()
	}
	if e.isInputModeLTOpened && e.inputModeLookupTable.CursorUp() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusBambooEngine) CursorDown() *dbus.Error {
	if e.isEmojiLTOpened && e.emojiLookupTable.CursorDown() {
		e.updateEmojiLookupTable()
	}
	if e.isInputModeLTOpened && e.inputModeLookupTable.CursorDown() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusBambooEngine) CandidateClicked(index uint32, button uint32, state uint32) *dbus.Error {
	if e.isEmojiLTOpened && e.updateCursorPosInEmojiTable(index) {
		e.commitEmojiCandidate()
		e.closeEmojiCandidates()
	}
	if e.isInputModeLTOpened && e.inputModeLookupTable.SetCursorPos(index) {
		e.commitInputModeCandidate()
		e.closeInputModeCandidates()
	}
	return nil
}

func (e *IBusBambooEngine) SetCapabilities(cap uint32) *dbus.Error {
	e.capabilities = cap
	return nil
}

func (e *IBusBambooEngine) SetCursorLocation(x int32, y int32, w int32, h int32) *dbus.Error {
	return nil
}

func (e *IBusBambooEngine) SetContentType(purpose uint32, hints uint32) *dbus.Error {
	return nil
}

// @method(in_signature="su")
func (e *IBusBambooEngine) PropertyActivate(propName string, propState uint32) *dbus.Error {
	if propName == PropKeyAbout {
		OpenIbusBambooPage()
		return nil
	}
	if propName == PropKeyVnCharsetConvert {
		OpenCharsetConvertPage()
		return nil
	}
	if propName == PropKeyConfiguration || propName == PropKeyInputModeLookupTableShortcut ||
		propName == PropKeyMacroTable {
		e.OpenConfigurationWindow()
	}

	if propName == PropKeyStdToneStyle {
		e.TurnStandardToneStyle(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyFreeToneMarking {
		e.TurnFreeToneMarking(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyEnableSpellCheck {
		e.TurnSpellChecking(propState == ibus.PROP_STATE_CHECKED)
	}

	if propName == PropKeySpellCheckByRules {
		e.TurnSpellCheckingByRules(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeySpellCheckByDicts {
		e.TurnSpellCheckingByDicts(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyMouseCapturing {
		e.TurnMouseCapturing(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyMacroEnabled {
		e.TurnMacroEnabled(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyPreeditInvisibility {
		e.TurnPreeditInvisibility(propState == ibus.PROP_STATE_CHECKED)
	}
	if propName == PropKeyPreeditElimination {
		if propState == ibus.PROP_STATE_CHECKED {
			e.config.IBflags |= config.IBpreeditElimination
		} else {
			e.config.IBflags &= ^config.IBpreeditElimination
		}
	}
	if propName == PropKeyAutoCapitalizeMacro {
		e.TurnAutoCapitalizeMacro(propState == ibus.PROP_STATE_CHECKED)
	}

	var inputMethodSelected = GetInputMethodSelectedFromMenu(propName)
	if inputMethodSelected != -1 {
		e.SetDefaultInputMethod(inputMethodSelected)
	}

	var outputCharsetSelected = GetOutputCharsetSelectedFromMenu(propName)
	if outputCharsetSelected != "" {
		e.config.OutputCharset = outputCharsetSelected
	}
	if e.IsInputMethodSelectedFromMenuExists(propName) {
		e.config.InputMethod = propName
	}
	if propName != "-" {
		config.SaveConfig(e.config, e.engineName)
	}
	e.propList = GetPropListByConfig(e.config)

	var inputMethod = bamboo.ParseInputMethod(e.config.InputMethodDefinitions, e.config.InputMethod)
	e.preeditor = bamboo.NewEngine(inputMethod, e.config.Flags)
	e.RegisterProperties(e.propList)
	return nil
}
