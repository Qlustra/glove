package driver

import (
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/qlustra/glove/internal/command"
)

var (
	binType           = reflect.TypeOf(Bin{})
	stringerType      = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

type schema struct {
	program     string
	declaredBin string
	root        *node
	nodes       map[reflect.Type]*node
}

type node struct {
	typ      reflect.Type
	cmd      string
	flags    []flagField
	args     []argField
	children map[reflect.Type]*node
}

type flagMode uint8

const (
	flagModeBool flagMode = iota + 1
	flagModeScalar
	flagModePointer
	flagModeSlice
	flagModeMap
)

type flagField struct {
	fieldName string
	index     []int
	name      string
	aliases   []string
	mode      flagMode
}

type argMode uint8

const (
	argModeScalar argMode = iota + 1
	argModePointer
	argModeTail
)

type argField struct {
	fieldName string
	index     []int
	position  int
	required  bool
	tail      bool
	mode      argMode
}

type compiler struct {
	nodes map[reflect.Type]*node
}

func Compile[T any](decl T, options ...CompileOptions) (Driver[T], error) {
	rootType, err := structTypeOf(reflect.TypeOf(decl))
	if err != nil {
		return Driver[T]{}, fmt.Errorf("compile driver: %w", err)
	}

	resolved, err := resolveCompileOptions(options)
	if err != nil {
		return Driver[T]{}, fmt.Errorf("compile driver: %w", err)
	}

	compiled, err := compileSchema(rootType, resolved)
	if err != nil {
		return Driver[T]{}, err
	}

	runner := resolved.Runner
	if runner == nil {
		runner = command.ExecRunner{}
	}

	return Driver[T]{
		compiled: compiled,
		runner:   runner,
	}, nil
}

func New[T any](decl T, options ...CompileOptions) Driver[T] {
	driver, err := Compile(decl, options...)
	if err != nil {
		panic(err)
	}
	return driver
}

func resolveCompileOptions(options []CompileOptions) (CompileOptions, error) {
	switch len(options) {
	case 0:
		return CompileOptions{}, nil
	case 1:
		return options[0], nil
	default:
		return CompileOptions{}, fmt.Errorf("expected at most one CompileOptions value")
	}
}

func compileSchema(rootType reflect.Type, options CompileOptions) (*schema, error) {
	c := compiler{
		nodes: make(map[reflect.Type]*node),
	}

	root, declaredBin, err := c.compileNode(rootType, true)
	if err != nil {
		return nil, err
	}

	program := declaredBin
	if options.Bin != "" {
		program = options.Bin
	}

	return &schema{
		program:     program,
		declaredBin: declaredBin,
		root:        root,
		nodes:       c.nodes,
	}, nil
}

func (c *compiler) compileNode(typ reflect.Type, root bool) (*node, string, error) {
	if _, exists := c.nodes[typ]; exists {
		return nil, "", fmt.Errorf("compile driver: frame type %s is reused in multiple command paths", typ)
	}

	n := &node{
		typ:      typ,
		children: make(map[reflect.Type]*node),
	}
	c.nodes[typ] = n

	var declaredBin string
	flagNames := make(map[string]string)
	childNames := make(map[string]string)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		binTag := strings.TrimSpace(field.Tag.Get("bin"))
		cmdTag := strings.TrimSpace(field.Tag.Get("cmd"))
		flagTag := strings.TrimSpace(field.Tag.Get("flag"))
		argTag := strings.TrimSpace(field.Tag.Get("arg"))

		tagCount := 0
		if binTag != "" {
			tagCount++
		}
		if cmdTag != "" {
			tagCount++
		}
		if flagTag != "" {
			tagCount++
		}
		if argTag != "" {
			tagCount++
		}

		if tagCount == 0 {
			continue
		}
		if field.PkgPath != "" {
			return nil, "", fmt.Errorf("compile driver: %s.%s is unexported but tagged", typ, field.Name)
		}
		if tagCount > 1 {
			return nil, "", fmt.Errorf("compile driver: %s.%s uses multiple glove tags", typ, field.Name)
		}

		switch {
		case binTag != "":
			if !root {
				return nil, "", fmt.Errorf("compile driver: %s.%s uses bin outside the driver root", typ, field.Name)
			}
			if field.Type != binType {
				return nil, "", fmt.Errorf("compile driver: %s.%s must be glove.Bin", typ, field.Name)
			}
			if declaredBin != "" {
				return nil, "", fmt.Errorf("compile driver: %s declares multiple bin fields", typ)
			}
			declaredBin = binTag

		case cmdTag != "":
			childType, err := structTypeOf(field.Type)
			if err != nil {
				return nil, "", fmt.Errorf("compile driver: %s.%s: %w", typ, field.Name, err)
			}
			if prior, exists := childNames[cmdTag]; exists {
				return nil, "", fmt.Errorf("compile driver: %s reuses command name %q on fields %s and %s", typ, cmdTag, prior, field.Name)
			}

			child, _, err := c.compileNode(childType, false)
			if err != nil {
				return nil, "", err
			}
			child.cmd = cmdTag
			n.children[childType] = child
			childNames[cmdTag] = field.Name

		case flagTag != "":
			flag, err := parseFlagField(field)
			if err != nil {
				return nil, "", fmt.Errorf("compile driver: %s.%s: %w", typ, field.Name, err)
			}

			for _, name := range append([]string{flag.name}, flag.aliases...) {
				if prior, exists := flagNames[name]; exists {
					return nil, "", fmt.Errorf("compile driver: %s reuses flag name %q on fields %s and %s", typ, name, prior, field.Name)
				}
				flagNames[name] = field.Name
			}

			n.flags = append(n.flags, flag)

		case argTag != "":
			arg, err := parseArgField(field)
			if err != nil {
				return nil, "", fmt.Errorf("compile driver: %s.%s: %w", typ, field.Name, err)
			}
			n.args = append(n.args, arg)
		}
	}

	if root && declaredBin == "" {
		return nil, "", fmt.Errorf("compile driver: %s is missing a Bin field with a bin tag", typ)
	}

	if err := validateArgs(typ, n.args); err != nil {
		return nil, "", err
	}

	return n, declaredBin, nil
}

func parseFlagField(field reflect.StructField) (flagField, error) {
	tokens := splitTag(field.Tag.Get("flag"))
	if len(tokens) == 0 {
		return flagField{}, fmt.Errorf("flag tag must declare at least one flag name")
	}

	var (
		names     []string
		sawRepeat bool
		sawKV     bool
	)

	for _, token := range tokens {
		switch token {
		case "repeat":
			sawRepeat = true
		case "kv":
			sawKV = true
		default:
			if !strings.HasPrefix(token, "-") {
				return flagField{}, fmt.Errorf("flag token %q must start with '-'", token)
			}
			names = append(names, token)
		}
	}

	if len(names) == 0 {
		return flagField{}, fmt.Errorf("flag tag must declare at least one flag name")
	}

	mode, err := classifyFlagType(field.Type)
	if err != nil {
		return flagField{}, err
	}
	if sawRepeat && mode != flagModeSlice && mode != flagModeMap {
		return flagField{}, fmt.Errorf("repeat is only valid on slices and maps")
	}
	if sawKV && mode != flagModeMap {
		return flagField{}, fmt.Errorf("kv is only valid on map[string]string")
	}

	return flagField{
		fieldName: field.Name,
		index:     append([]int(nil), field.Index...),
		name:      names[0],
		aliases:   names[1:],
		mode:      mode,
	}, nil
}

func parseArgField(field reflect.StructField) (argField, error) {
	tokens := splitTag(field.Tag.Get("arg"))
	if len(tokens) == 0 {
		return argField{}, fmt.Errorf("arg tag must start with a position")
	}

	position, err := strconv.Atoi(tokens[0])
	if err != nil || position < 0 {
		return argField{}, fmt.Errorf("arg position %q must be a non-negative integer", tokens[0])
	}

	var (
		required bool
		tail     bool
	)

	for _, token := range tokens[1:] {
		switch token {
		case "required":
			required = true
		case "tail":
			tail = true
		default:
			return argField{}, fmt.Errorf("arg option %q is not supported", token)
		}
	}

	mode, err := classifyArgType(field.Type, tail)
	if err != nil {
		return argField{}, err
	}

	return argField{
		fieldName: field.Name,
		index:     append([]int(nil), field.Index...),
		position:  position,
		required:  required,
		tail:      tail,
		mode:      mode,
	}, nil
}

func classifyFlagType(typ reflect.Type) (flagMode, error) {
	switch {
	case typ.Kind() == reflect.Bool:
		return flagModeBool, nil

	case typ.Kind() == reflect.Pointer:
		if typ.Elem().Kind() == reflect.Bool {
			return 0, fmt.Errorf("pointer bool flags are not supported")
		}
		if !isScalarType(typ.Elem()) {
			return 0, fmt.Errorf("unsupported flag type %s", typ)
		}
		return flagModePointer, nil

	case typ.Kind() == reflect.Slice:
		if typ.Elem().Kind() == reflect.Bool {
			return 0, fmt.Errorf("slice bool flags are not supported")
		}
		if !isScalarType(typ.Elem()) {
			return 0, fmt.Errorf("unsupported flag slice type %s", typ)
		}
		return flagModeSlice, nil

	case typ.Kind() == reflect.Map:
		if typ.Key().Kind() != reflect.String || typ.Elem().Kind() != reflect.String {
			return 0, fmt.Errorf("map flags must be map[string]string, got %s", typ)
		}
		return flagModeMap, nil

	case isScalarType(typ):
		return flagModeScalar, nil
	}

	return 0, fmt.Errorf("unsupported flag type %s", typ)
}

func classifyArgType(typ reflect.Type, tail bool) (argMode, error) {
	if tail {
		if typ.Kind() != reflect.Slice || typ.Elem().Kind() != reflect.String {
			return 0, fmt.Errorf("tail args must be []string, got %s", typ)
		}
		return argModeTail, nil
	}

	switch {
	case typ.Kind() == reflect.Pointer:
		if typ.Elem().Kind() == reflect.Bool {
			return 0, fmt.Errorf("pointer bool args are not supported")
		}
		if !isScalarType(typ.Elem()) {
			return 0, fmt.Errorf("unsupported arg type %s", typ)
		}
		return argModePointer, nil

	case typ.Kind() == reflect.Bool:
		return 0, fmt.Errorf("bool args are not supported")

	case isScalarType(typ):
		return argModeScalar, nil
	}

	return 0, fmt.Errorf("unsupported arg type %s", typ)
}

func validateArgs(typ reflect.Type, args []argField) error {
	if len(args) == 0 {
		return nil
	}

	sort.Slice(args, func(i, j int) bool {
		return args[i].position < args[j].position
	})

	seenOptional := false
	for i, arg := range args {
		if arg.position != i {
			return fmt.Errorf("compile driver: %s positional args must be contiguous starting at 0", typ)
		}
		if arg.tail && i != len(args)-1 {
			return fmt.Errorf("compile driver: %s.%s tail arg must be last", typ, arg.fieldName)
		}
		if seenOptional && arg.required && !arg.tail {
			return fmt.Errorf("compile driver: %s.%s cannot be required after an optional positional arg", typ, arg.fieldName)
		}
		if !arg.required && !arg.tail {
			seenOptional = true
		}
	}

	return nil
}

func structTypeOf(typ reflect.Type) (reflect.Type, error) {
	if typ == nil {
		return nil, fmt.Errorf("type must not be nil")
	}

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type must be a struct or pointer to struct, got %s", typ)
	}

	return typ, nil
}

func splitTag(raw string) []string {
	parts := strings.Split(raw, ",")
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func isScalarType(typ reflect.Type) bool {
	if typ == nil {
		return false
	}

	if typ.Implements(textMarshalerType) || typ.Implements(stringerType) {
		return true
	}
	if typ.Kind() != reflect.Pointer {
		ptr := reflect.PointerTo(typ)
		if ptr.Implements(textMarshalerType) || ptr.Implements(stringerType) {
			return true
		}
	}

	switch typ.Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
