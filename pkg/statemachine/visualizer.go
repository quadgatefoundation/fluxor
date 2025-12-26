package statemachine

import (
	"fmt"
	"strings"
)

// Visualizer generates visual representations of state machines.
type Visualizer struct {
	definition *Definition
}

// NewVisualizer creates a new visualizer for a state machine definition.
func NewVisualizer(def *Definition) *Visualizer {
	return &Visualizer{definition: def}
}

// ToMermaid generates a Mermaid diagram of the state machine.
func (v *Visualizer) ToMermaid() string {
	var sb strings.Builder
	
	sb.WriteString("```mermaid\n")
	sb.WriteString("stateDiagram-v2\n")
	
	// Start state
	sb.WriteString(fmt.Sprintf("    [*] --> %s\n", v.definition.InitialState))
	
	// States and transitions
	for _, state := range v.definition.States {
		// Mark final states
		if state.IsFinal {
			sb.WriteString(fmt.Sprintf("    %s --> [*]\n", state.Name))
		}
		
		// Transitions
		for _, transition := range state.Transitions {
			label := transition.Event
			if transition.Guard != nil {
				label += " [guarded]"
			}
			sb.WriteString(fmt.Sprintf("    %s --> %s : %s\n", 
				state.Name, transition.To, label))
		}
	}
	
	sb.WriteString("```\n")
	return sb.String()
}

// ToASCII generates an ASCII diagram of the state machine.
func (v *Visualizer) ToASCII() string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("State Machine: %s\n", v.definition.Name))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")
	
	sb.WriteString(fmt.Sprintf("Initial State: %s\n\n", v.definition.InitialState))
	
	// List all states
	sb.WriteString("States:\n")
	for name, state := range v.definition.States {
		finalMarker := ""
		if state.IsFinal {
			finalMarker = " (final)"
		}
		sb.WriteString(fmt.Sprintf("  • %s%s\n", name, finalMarker))
		
		// List transitions
		if len(state.Transitions) > 0 {
			for _, t := range state.Transitions {
				guardMarker := ""
				if t.Guard != nil {
					guardMarker = " [guarded]"
				}
				actionMarker := ""
				if t.Action != nil {
					actionMarker = " [action]"
				}
				sb.WriteString(fmt.Sprintf("      %s → %s%s%s\n", 
					t.Event, t.To, guardMarker, actionMarker))
			}
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// ToGraphviz generates a Graphviz DOT representation.
func (v *Visualizer) ToGraphviz() string {
	var sb strings.Builder
	
	sb.WriteString("digraph StateMachine {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=circle];\n\n")
	
	// Start node
	sb.WriteString("  start [shape=point];\n")
	sb.WriteString(fmt.Sprintf("  start -> %s;\n\n", v.definition.InitialState))
	
	// States
	for name, state := range v.definition.States {
		shape := "circle"
		if state.IsFinal {
			shape = "doublecircle"
		}
		sb.WriteString(fmt.Sprintf("  %s [shape=%s];\n", name, shape))
		
		// Transitions
		for _, t := range state.Transitions {
			label := t.Event
			if t.Guard != nil {
				label += "\\n[guard]"
			}
			if t.Action != nil {
				label += "\\n[action]"
			}
			sb.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\"];\n", 
				name, t.To, label))
		}
	}
	
	sb.WriteString("}\n")
	return sb.String()
}

// ToJSON generates a JSON representation suitable for visualization tools.
func (v *Visualizer) ToJSON() string {
	// Build nodes
	var nodes []string
	for name, state := range v.definition.States {
		nodeType := "normal"
		if name == v.definition.InitialState {
			nodeType = "initial"
		} else if state.IsFinal {
			nodeType = "final"
		}
		
		nodes = append(nodes, fmt.Sprintf(`{"id":"%s","type":"%s"}`, name, nodeType))
	}
	
	// Build edges
	var edges []string
	for _, state := range v.definition.States {
		for _, t := range state.Transitions {
			hasGuard := t.Guard != nil
			hasAction := t.Action != nil
			edges = append(edges, fmt.Sprintf(
				`{"from":"%s","to":"%s","event":"%s","guarded":%t,"hasAction":%t}`,
				state.Name, t.To, t.Event, hasGuard, hasAction))
		}
	}
	
	return fmt.Sprintf(`{"nodes":[%s],"edges":[%s]}`, 
		strings.Join(nodes, ","), strings.Join(edges, ","))
}

// GetStats returns statistics about the state machine.
func (v *Visualizer) GetStats() map[string]interface{} {
	stateCount := len(v.definition.States)
	transitionCount := 0
	finalStateCount := 0
	
	for _, state := range v.definition.States {
		transitionCount += len(state.Transitions)
		if state.IsFinal {
			finalStateCount++
		}
	}
	
	return map[string]interface{}{
		"id":               v.definition.ID,
		"name":             v.definition.Name,
		"initialState":     v.definition.InitialState,
		"stateCount":       stateCount,
		"transitionCount":  transitionCount,
		"finalStateCount":  finalStateCount,
	}
}

// Validate performs validation and returns a list of warnings/issues.
func (v *Visualizer) Validate() []string {
	var issues []string
	
	// Check for unreachable states
	reachable := make(map[string]bool)
	reachable[v.definition.InitialState] = true
	
	// BFS to find all reachable states
	queue := []string{v.definition.InitialState}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		state := v.definition.States[current]
		if state != nil {
			for _, t := range state.Transitions {
				if !reachable[t.To] {
					reachable[t.To] = true
					queue = append(queue, t.To)
				}
			}
		}
	}
	
	for name := range v.definition.States {
		if !reachable[name] {
			issues = append(issues, fmt.Sprintf("State '%s' is unreachable", name))
		}
	}
	
	// Check for states with no outgoing transitions (potential dead ends)
	for name, state := range v.definition.States {
		if len(state.Transitions) == 0 && !state.IsFinal {
			issues = append(issues, fmt.Sprintf("State '%s' has no outgoing transitions and is not marked as final", name))
		}
	}
	
	// Check for duplicate transitions
	for name, state := range v.definition.States {
		events := make(map[string]int)
		for _, t := range state.Transitions {
			events[t.Event]++
		}
		for event, count := range events {
			if count > 1 {
				issues = append(issues, fmt.Sprintf("State '%s' has %d transitions for event '%s' (consider using priorities)", name, count, event))
			}
		}
	}
	
	return issues
}
