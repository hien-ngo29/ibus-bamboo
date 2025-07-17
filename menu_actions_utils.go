/*
 * Bamboo - A Vietnamese Input method editor
 * Copyright (C) 2025 Ngo Phu Hien <ngophuhien029@gmail.com>
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
	"ibus-bamboo/config"
	"ibus-bamboo/ui"
	"os/exec"
	"strconv"

	"github.com/BambooEngine/bamboo-core"
)

func OpenIbusBambooPage() {
	exec.Command("xdg-open", HomePage).Start()
}

// Chuyen ma Online
func OpenCharsetConvertPage() {
	exec.Command("xdg-open", CharsetConvertPage).Start()
}

func (e *IBusBambooEngine) OpenConfigurationWindow() {
	ui.OpenGUI(e.engineName)
	e.config = config.LoadConfig(e.engineName)
}

// Bat kiem tra chinh ta
func (e *IBusBambooEngine) TurnSpellChecking(on bool) {
	if on {
		e.config.IBflags |= config.IBspellCheckEnabled
		e.config.IBflags |= config.IBautoNonVnRestore

		if e.config.IBflags&config.IBspellCheckWithDicts == 0 {
			e.config.IBflags |= config.IBspellCheckWithRules
		}

	} else {
		e.config.IBflags &= ^config.IBspellCheckEnabled
		e.config.IBflags &= ^config.IBautoNonVnRestore
	}
}

// Dau thanh chuan
func (e *IBusBambooEngine) TurnStandardToneStyle(on bool) {
	if on {
		e.config.Flags |= bamboo.EstdToneStyle
	} else {
		e.config.Flags &= ^bamboo.EstdToneStyle
	}
}

// Bo dau tu do
func (e *IBusBambooEngine) TurnFreeToneMarking(on bool) {
	if on {
		e.config.Flags |= bamboo.EfreeToneMarking
	} else {
		e.config.Flags &= ^bamboo.EfreeToneMarking
	}
}

// Su dung luat ghep van
func (e *IBusBambooEngine) TurnSpellCheckingByRules(on bool) {
	if on {
		e.config.IBflags |= config.IBspellCheckWithRules
		e.TurnSpellChecking(true)
	} else {
		e.config.IBflags &= ^config.IBspellCheckWithRules
	}
}

// Su dung tu dien
func (e *IBusBambooEngine) TurnSpellCheckingByDicts(on bool) {
	if on {
		e.config.IBflags |= config.IBspellCheckWithDicts
		e.TurnSpellChecking(true)
		dictionary, _ = loadDictionary(DictVietnameseCm)
	} else {
		e.config.IBflags &= ^config.IBspellCheckWithDicts
	}
}

// Bat su kien chuot
func (e *IBusBambooEngine) TurnMouseCapturing(on bool) {
	if on {
		e.config.IBflags |= config.IBmouseCapturing
		startMouseCapturing()
		startMouseRecording()
	} else {
		e.config.IBflags &= ^config.IBmouseCapturing
		stopMouseCapturing()
		stopMouseRecording()
	}
}

// Bat go tat
func (e *IBusBambooEngine) TurnMacroEnabled(on bool) {
	if on {
		e.config.IBflags |= config.IBmacroEnabled
		e.macroTable.Enable(e.engineName)
	} else {
		e.config.IBflags &= ^config.IBmacroEnabled
		e.macroTable.Disable()
	}
}

// An gach chan
func (e *IBusBambooEngine) TurnPreeditInvisibility(on bool) {
	if on {
		e.config.IBflags |= config.IBnoUnderline
	} else {
		e.config.IBflags &= ^config.IBnoUnderline
	}
}

// TODO: Add PropKeyPreeditElimination

// Tu dong viet hoa
func (e *IBusBambooEngine) TurnAutoCapitalizeMacro(on bool) {
	if on {
		e.config.IBflags |= config.IBautoCapitalizeMacro
	} else {
		e.config.IBflags &= ^config.IBautoCapitalizeMacro
	}
	if e.config.IBflags&config.IBmacroEnabled != 0 {
		e.macroTable.Reload(e.engineName, e.config.IBflags&config.IBautoCapitalizeMacro != 0)
	}
}

func GetInputMethodSelectedFromMenu(propName string) int {
	var im, foundIm = getValueFromPropKey(propName, "InputMode")
	if foundIm {
		var inputMode, _ = strconv.Atoi(im)
		return inputMode
	}

	return -1
}

func (e *IBusBambooEngine) SetDefaultInputMethod(inputMethod int) {
	e.config.DefaultInputMode = inputMethod
}

func GetOutputCharsetSelectedFromMenu(propName string) string {
	var charset, foundCs = getValueFromPropKey(propName, "OutputCharset")
	if foundCs && isValidCharset(charset) {
		return charset
	}
	return ""
}

func (e *IBusBambooEngine) IsInputMethodSelectedFromMenuExists(propName string) bool {
	var _, found = e.config.InputMethodDefinitions[propName]
	return found
}
