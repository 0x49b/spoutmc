import React, { useRef, useState } from 'react';
import { Menu, MenuContent, MenuItem, MenuList, Popper, TextInput } from '@patternfly/react-core';
import { commandData } from '@app/Server/details/components/commandAutocomplete/commandData';

type ParsedArg = {
  name: string
  type: string
  suggestions?: string[]
}

type ValidState = 'success' | 'error' | 'default'

type CommandAutocompleteProps = {
  onComplete: (command: string) => void
}

const parseArgument = (arg: string | Record<string, string[]>): ParsedArg => {
  if (typeof arg === 'string') {
    const match = arg.match(/^<([\w-]+):\s*(.+)>$/)
    if (match) {
      return { name: match[1], type: match[2] }
    }
  } else if (typeof arg === 'object') {
    const [key] = Object.keys(arg)
    const match = key.match(/^<([\w-]+):\s*(.+)>$/)
    if (match) {
      return {
        name: match[1],
        type: match[2],
        suggestions: arg[key].map(String)
      }
    }
  }
  return { name: '', type: 'string' }
}

const getSuggestions = (input: string): { suggestions: string[]; argIndex: number } => {
  const parts = input.trim().split(/\s+/)
  const [command, ...args] = parts

  if (!command) {
    return { suggestions: commandData.map(cmd => Object.keys(cmd)[0]), argIndex: 0 }
  }

  const entry = commandData.find(cmd => cmd[command])
  if (!entry) return { suggestions: [], argIndex: 0 }

  const commandArgs = entry[command]
  const currentArgIndex = args.length === 0 ? 0 : input.endsWith(' ') ? args.length : args.length - 1

  if (currentArgIndex >= commandArgs.length) {
    return { suggestions: [], argIndex: currentArgIndex }
  }

  const argSpec = commandArgs[currentArgIndex]
  const parsed = parseArgument(argSpec)

  if (Array.isArray(argSpec)) {
    const currentInput = args[currentArgIndex - 1] || ''
    const options = (argSpec as string[]).map(String)
    return {
      suggestions: options.filter(opt => opt.toLowerCase().startsWith(currentInput.toLowerCase())),
      argIndex: currentArgIndex
    }
  }

  if (parsed.suggestions) {
    const currentInput = args[currentArgIndex] || ''
    return {
      suggestions: parsed.suggestions.filter(val =>
        val.toLowerCase().startsWith(currentInput.toLowerCase())
      ),
      argIndex: currentArgIndex
    }
  }

  if (parsed.type.includes('|')) {
    const types = parsed.type.split('|').map(s => s.trim())
    if (types.includes('bool')) {
      return { suggestions: ['true', 'false'], argIndex: currentArgIndex }
    }
  }

  return { suggestions: [], argIndex: currentArgIndex }
}

const isValidCommand = (value: string): boolean => {
  const parts = value.trim().split(/\s+/)
  const [command, ...args] = parts

  const entry = commandData.find(cmd => cmd[command])
  if (!entry) return false

  const expectedArgs = entry[command]
  if (args.length !== expectedArgs.length) return false

  for (let i = 0; i < args.length; i++) {
    const expectedArg = expectedArgs[i]
    const parsed = parseArgument(expectedArg)
    const inputVal = args[i]

    if (parsed.type === 'x y z') {
      const coords = inputVal.split(' ')
      if (coords.length !== 3 || coords.some(v => isNaN(Number(v)))) {
        return false
      }
    }

    if (parsed.type === 'bool') {
      if (inputVal !== 'true' && inputVal !== 'false') {
        return false
      }
    }

    if (parsed.type === 'int') {
      if (isNaN(Number(inputVal))) {
        return false
      }
    }

    if (parsed.type.includes('|')) {
      const types = parsed.type.split('|').map(s => s.trim())
      if (types.includes('bool') && (inputVal === 'true' || inputVal === 'false')) {
        continue
      } else if (types.includes('int') && !isNaN(Number(inputVal))) {
        continue
      }
      return false
    }

    if (parsed.suggestions && !parsed.suggestions.includes(inputVal)) {
      return false
    }
  }

  return true
}

const CommandAutocomplete: React.FC<CommandAutocompleteProps> = ({ onComplete }) => {
  const [input, setInput] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [isOpen, setIsOpen] = useState(false)
  const [validState, setValidState] = useState<ValidState>('default')
  const inputRef = useRef<HTMLInputElement>(null)

  const updateSuggestions = (value: string) => {
    const { suggestions } = getSuggestions(value)
    setSuggestions(suggestions)
    setSelectedIndex(0)
    setIsOpen(suggestions.length > 0)
  }

  const triggerValidation = (value: string) => {
    if (isValidCommand(value)) {
      setValidState('success')
      setIsOpen(false)
      onComplete(value.trim())
    } else {
      setValidState('error')
    }
  }

  const handleInputChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
    setInput(value)
    setValidState('default')
    updateSuggestions(value)
  }

  const handleFocus = () => {
    updateSuggestions(input)
  }

  const handleSelect = (value: string) => {
    const parts = input.trim().split(/\s+/)
    const { argIndex } = getSuggestions(input)

    const updatedParts = [...parts]
    updatedParts[argIndex + 1] = value // +1 to skip command
    const commandName = parts[0]
    const entry = commandData.find(cmd => cmd[commandName])
    const maxArgs = entry ? entry[commandName].length : 0

    let newInput = [commandName, ...updatedParts.slice(1, argIndex + 2)].join(' ')
    if (argIndex + 1 < maxArgs) {
      newInput += ' '
    }

    setInput(newInput)
    updateSuggestions(newInput)

    if ((argIndex + 1) === maxArgs && isValidCommand(newInput)) {
      setValidState('success')
      setIsOpen(false)
      onComplete(newInput.trim())
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (isOpen) {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex((prev) => (prev + 1) % suggestions.length)
          return
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex((prev) => (prev - 1 + suggestions.length) % suggestions.length)
          return
        case 'Enter':
          e.preventDefault()
          handleSelect(suggestions[selectedIndex])
          return
        case 'Escape':
          e.preventDefault()
          setIsOpen(false)
          return
      }
    }

    if (e.key === 'Enter') {
      triggerValidation(input)
    }
  }

  return (
    <div style={{ position: 'relative', width: '400px' }}>
      <TextInput
        ref={inputRef}
        value={input}
        type="text"
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        onFocus={handleFocus}
        onBlur={() => triggerValidation(input)}
        aria-label="Command input"
        placeholder="Enter command..."
        validated={validState}
      />
      <Popper
        triggerRef={inputRef}
        isVisible={isOpen}
        appendTo={document.body}
        popper={
          <Menu>
            <MenuContent>
              <MenuList>
                {suggestions.map((s, i) => (
                  <MenuItem
                    key={s}
                    isSelected={i === selectedIndex}
                    onClick={() => handleSelect(s)}
                  >
                    {s}
                  </MenuItem>
                ))}
              </MenuList>
            </MenuContent>
          </Menu>
        }
      />
    </div>
  )
}

export default CommandAutocomplete
