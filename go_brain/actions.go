package main

import (
	"fmt"
)

type Action struct {
	Type   string      `json:"type"`
	Target string      `json:"target"`
	Value  interface{} `json:"value"`
}

type ActionExecutor struct {
	dendrite *Dendrite
}

func (e *ActionExecutor) Execute(action Action) error {
	infoLog.Printf("ACTUATING: [%s] -> %s", action.Type, action.Target)
	switch action.Type {
	case "move":
		return e.handleMove(action.Target, action.Value)
	case "look":
		return e.handleLook(action.Target)
	case "speak":
		return speak(action.Target)
	case "remember":
		e.dendrite.Upsert(toNodeID(action.Target), action.Target, fmt.Sprintf("%v", action.Value), NodeTypeConcept, []string{"learned"})
		return nil
	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

func (e *ActionExecutor) handleMove(dir string, speed interface{}) error {
	s := byte(150)
	if val, ok := speed.(float64); ok {
		s = byte(val)
	}
	switch dir {
	case "forward":
		for i := byte(0); i < 4; i++ {
			moveMotor(i, 0, s)
		}
	case "backward":
		for i := byte(0); i < 4; i++ {
			moveMotor(i, 1, s)
		}
	case "stop":
		stopAllMotors()
	}
	// Motors do NOT auto-stop - caller must send stop
	return nil
}

func (e *ActionExecutor) handleLook(dir string) error {
	switch dir {
	case "up":
		return setServo(2, 170)
	case "down":
		return setServo(2, 30)
	case "center":
		_ = setServo(1, 90)
		return setServo(2, 110)
	}
	return nil
}
