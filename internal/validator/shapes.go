package validator

import "fmt"

type ShapeLimits struct {
	MinOCPU   float64
	MaxOCPU   float64
	MinMemory float64
	MaxMemory float64
}

var supportedShapes = map[string]ShapeLimits{
	"VM.Standard.E4.Flex": {MinOCPU: 1, MaxOCPU: 32, MinMemory: 1, MaxMemory: 512},
	"VM.Standard.E3.Flex": {MinOCPU: 1, MaxOCPU: 32, MinMemory: 1, MaxMemory: 512},
	"VM.Standard.A1.Flex": {MinOCPU: 1, MaxOCPU: 80, MinMemory: 1, MaxMemory: 512},
	"VM.Standard3.Flex":   {MinOCPU: 1, MaxOCPU: 32, MinMemory: 1, MaxMemory: 512},
}

const (
	MinBootVolumeGB = 10
	MaxBootVolumeGB = 200
)

func ValidateShape(shape string) error {
	if _, ok := supportedShapes[shape]; !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	return nil
}

func ValidateOCPU(shape string, ocpu float64) error {
	limits, ok := supportedShapes[shape]
	if !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	if ocpu < limits.MinOCPU || ocpu > limits.MaxOCPU {
		return fmt.Errorf("ocpu %.1f out of range [%.0f, %.0f] for shape %s", ocpu, limits.MinOCPU, limits.MaxOCPU, shape)
	}
	return nil
}

func ValidateMemory(shape string, memory float64) error {
	limits, ok := supportedShapes[shape]
	if !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	if memory < limits.MinMemory || memory > limits.MaxMemory {
		return fmt.Errorf("memory %.1f GB out of range [%.0f, %.0f] for shape %s", memory, limits.MinMemory, limits.MaxMemory, shape)
	}
	return nil
}

func ValidateBootVolume(gb int) error {
	if gb < MinBootVolumeGB || gb > MaxBootVolumeGB {
		return fmt.Errorf("boot volume %d GB out of range [%d, %d]", gb, MinBootVolumeGB, MaxBootVolumeGB)
	}
	return nil
}

func SupportedShapes() []string {
	shapes := make([]string, 0, len(supportedShapes))
	for s := range supportedShapes {
		shapes = append(shapes, s)
	}
	return shapes
}
