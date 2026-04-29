package driver

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/qlustra/glove/internal/command"
)

func (d Driver[T]) Command(frames ...any) (command.Command, error) {
	if d.compiled == nil {
		return command.Command{}, fmt.Errorf("driver is not compiled")
	}

	cmd := d.baseCommand()
	current := d.compiled.root
	start := 0

	if len(frames) > 0 {
		frameType, frameValue, err := normalizeFrame(frames[0])
		if err != nil {
			return command.Command{}, err
		}
		if frameType == current.typ {
			rendered, err := renderNode(current, frameValue)
			if err != nil {
				return command.Command{}, err
			}
			cmd.Args = append(cmd.Args, rendered...)
			start = 1
		}
	}

	for i := start; i < len(frames); i++ {
		frameType, frameValue, err := normalizeFrame(frames[i])
		if err != nil {
			return command.Command{}, err
		}

		next, err := d.nextNode(current, frameType)
		if err != nil {
			return command.Command{}, err
		}

		cmd.Args = append(cmd.Args, next.cmd)

		rendered, err := renderNode(next, frameValue)
		if err != nil {
			return command.Command{}, err
		}
		cmd.Args = append(cmd.Args, rendered...)
		current = next
	}

	return cmd, nil
}

func (d Driver[T]) Run(ctx context.Context, frames ...any) command.Result {
	return d.RunWith(ctx, command.RunOptions{}, frames...)
}

func (d Driver[T]) RunWith(ctx context.Context, options command.RunOptions, frames ...any) command.Result {
	cmd, err := d.Command(frames...)
	if err != nil {
		return command.Result{
			Command: d.baseCommand(),
			Err:     err,
		}
	}

	if options.Runner == nil {
		options.Runner = d.runner
	}

	return cmd.RunWith(ctx, options)
}

func (d Driver[T]) With(ctx context.Context, options command.RunOptions, frames ...any) command.Result {
	return d.RunWith(ctx, options, frames...)
}

func (d Driver[T]) baseCommand() command.Command {
	if d.compiled == nil {
		return command.Command{}
	}

	return command.Command{
		Program: d.compiled.program,
	}
}

func (d Driver[T]) nextNode(current *node, frameType reflect.Type) (*node, error) {
	if next, ok := current.children[frameType]; ok {
		return next, nil
	}

	if _, known := d.compiled.nodes[frameType]; known {
		if current == d.compiled.root {
			return nil, fmt.Errorf("frame %s is not a root command of %s", frameType, d.compiled.root.typ)
		}
		return nil, fmt.Errorf("frame %s does not follow %s", frameType, current.typ)
	}

	return nil, fmt.Errorf("frame %s does not belong to driver %s", frameType, d.compiled.root.typ)
}

func normalizeFrame(frame any) (reflect.Type, reflect.Value, error) {
	if frame == nil {
		return nil, reflect.Value{}, fmt.Errorf("frame must not be nil")
	}

	value := reflect.ValueOf(frame)
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, reflect.Value{}, fmt.Errorf("frame must not be nil")
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, reflect.Value{}, fmt.Errorf("frame must be a struct or pointer to struct, got %T", frame)
	}

	value = addressableValue(value)
	return value.Type(), value, nil
}

func addressableValue(value reflect.Value) reflect.Value {
	if value.CanAddr() {
		return value
	}

	copy := reflect.New(value.Type()).Elem()
	copy.Set(value)
	return copy
}

func renderNode(n *node, value reflect.Value) ([]string, error) {
	args := make([]string, 0, len(n.flags)*2+len(n.args))

	for _, field := range n.flags {
		rendered, err := renderFlagField(field, value.FieldByIndex(field.index))
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", n.typ, field.fieldName, err)
		}
		args = append(args, rendered...)
	}

	positional, err := renderArgFields(n, value)
	if err != nil {
		return nil, err
	}
	args = append(args, positional...)

	return args, nil
}

func renderFlagField(field flagField, value reflect.Value) ([]string, error) {
	switch field.mode {
	case flagModeBool:
		if !value.Bool() {
			return nil, nil
		}
		return []string{field.name}, nil

	case flagModeScalar:
		if value.IsZero() {
			return nil, nil
		}
		rendered, err := stringifyScalar(value)
		if err != nil {
			return nil, err
		}
		return []string{field.name, rendered}, nil

	case flagModePointer:
		if value.IsNil() {
			return nil, nil
		}
		rendered, err := stringifyScalar(value.Elem())
		if err != nil {
			return nil, err
		}
		return []string{field.name, rendered}, nil

	case flagModeSlice:
		if value.Len() == 0 {
			return nil, nil
		}

		rendered := make([]string, 0, value.Len()*2)
		for i := 0; i < value.Len(); i++ {
			item, err := stringifyScalar(value.Index(i))
			if err != nil {
				return nil, err
			}
			rendered = append(rendered, field.name, item)
		}
		return rendered, nil

	case flagModeMap:
		if value.Len() == 0 {
			return nil, nil
		}

		keys := make([]string, 0, value.Len())
		for _, key := range value.MapKeys() {
			keys = append(keys, key.String())
		}
		sort.Strings(keys)

		rendered := make([]string, 0, len(keys)*2)
		keyType := value.Type().Key()
		for _, key := range keys {
			keyValue := reflect.New(keyType).Elem()
			keyValue.SetString(key)
			item := value.MapIndex(keyValue)
			rendered = append(rendered, field.name, key+"="+item.String())
		}
		return rendered, nil

	default:
		return nil, fmt.Errorf("unsupported flag mode %d", field.mode)
	}
}

func renderArgFields(n *node, value reflect.Value) ([]string, error) {
	args := make([]string, 0, len(n.args))
	seenGap := false

	for _, field := range n.args {
		rendered, present, err := renderArgField(field, value.FieldByIndex(field.index))
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", n.typ, field.fieldName, err)
		}

		if !present {
			if field.required {
				return nil, fmt.Errorf("%s.%s is required", n.typ, field.fieldName)
			}
			seenGap = true
			continue
		}
		if seenGap {
			return nil, fmt.Errorf("%s.%s cannot be set after an omitted positional argument", n.typ, field.fieldName)
		}

		args = append(args, rendered...)
	}

	return args, nil
}

func renderArgField(field argField, value reflect.Value) ([]string, bool, error) {
	switch field.mode {
	case argModeScalar:
		if value.IsZero() {
			return nil, false, nil
		}
		rendered, err := stringifyScalar(value)
		if err != nil {
			return nil, false, err
		}
		return []string{rendered}, true, nil

	case argModePointer:
		if value.IsNil() {
			return nil, false, nil
		}
		rendered, err := stringifyScalar(value.Elem())
		if err != nil {
			return nil, false, err
		}
		return []string{rendered}, true, nil

	case argModeTail:
		if value.Len() == 0 {
			return nil, false, nil
		}
		rendered := make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			rendered[i] = value.Index(i).String()
		}
		return rendered, true, nil

	default:
		return nil, false, fmt.Errorf("unsupported arg mode %d", field.mode)
	}
}

func stringifyScalar(value reflect.Value) (string, error) {
	value = addressableValue(value)

	if text, ok, err := marshalText(value); ok {
		return text, err
	}
	if text, ok := callStringer(value); ok {
		return text, nil
	}

	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'g', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'g', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported scalar type %s", value.Type())
	}
}

func marshalText(value reflect.Value) (string, bool, error) {
	if value.Type().Implements(textMarshalerType) {
		out, err := value.Interface().(encoding.TextMarshaler).MarshalText()
		return string(out), true, err
	}
	if value.CanAddr() {
		ptr := value.Addr()
		if ptr.Type().Implements(textMarshalerType) {
			out, err := ptr.Interface().(encoding.TextMarshaler).MarshalText()
			return string(out), true, err
		}
	}
	return "", false, nil
}

func callStringer(value reflect.Value) (string, bool) {
	if value.Type().Implements(stringerType) {
		return value.Interface().(fmt.Stringer).String(), true
	}
	if value.CanAddr() {
		ptr := value.Addr()
		if ptr.Type().Implements(stringerType) {
			return ptr.Interface().(fmt.Stringer).String(), true
		}
	}
	return "", false
}
