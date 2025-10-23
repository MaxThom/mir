package swarm_srvc

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

var (
	// Define the environment type for compilation
	// This tells expr what variables and functions are available
	env = map[string]any{
		// "t":     0.0, // elapsed time in seconds (will be set at runtime)
		// "time":  0.0, // alias for t (will be set at runtime)
		"sin":   math.Sin,
		"cos":   math.Cos,
		"tan":   math.Tan,
		"abs":   math.Abs,
		"sqrt":  math.Sqrt,
		"pow":   math.Pow,
		"exp":   math.Exp,
		"log":   math.Log,
		"log10": math.Log10,
		"floor": math.Floor,
		"ceil":  math.Ceil,
		"round": math.Round,
		"min":   math.Min,
		"max":   math.Max,
		"pi":    math.Pi,
		"π":     math.Pi,
		"e":     math.E,
		"rand":  Rand,
	}
)

// Generator holds a compiled expression and generates values
type Generator struct {
	program   *vm.Program
	startTime time.Time
}

// NewGenerator creates and compiles a new generator from an expression
func NewGenerator(tlmGen *mir_v1.SwarmTelemetryGenerator) (*Generator, error) {
	if tlmGen == nil || tlmGen.Expr == "" {
		return nil, fmt.Errorf("generator expression is empty")
	}

	// Compile the expression once - this validates syntax and creates bytecode
	program, err := expr.Compile(tlmGen.Expr, expr.Env(env), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", tlmGen.Expr, err)
	}

	return &Generator{
		program:   program,
		startTime: time.Now().UTC(),
	}, nil
}

// Generate evaluates the compiled expression and returns a value
func (g *Generator) Generate(now time.Time) (float64, error) {
	elapsed := now.Sub(g.startTime).Seconds()

	// Build runtime environment with current dynamic values
	// Only runtime variables change, functions are accessed from compilation env
	localEnv := map[string]any{
		"t":     elapsed,
		"x":     elapsed,
		"sin":   math.Sin,
		"cos":   math.Cos,
		"tan":   math.Tan,
		"abs":   math.Abs,
		"sqrt":  math.Sqrt,
		"pow":   math.Pow,
		"exp":   math.Exp,
		"log":   math.Log,
		"log10": math.Log10,
		"floor": math.Floor,
		"ceil":  math.Ceil,
		"round": math.Round,
		"min":   math.Min,
		"max":   math.Max,
		"pi":    math.Pi,
		"π":     math.Pi,
		"e":     math.E,
		"rand":  Rand,
	}

	// Run the compiled program with current environment
	output, err := vm.Run(g.program, localEnv)
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return toFloat64(output)
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("expression returned non-numeric type: %T", v)
	}
}

func Rand(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}
