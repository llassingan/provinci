package validator

import "fmt"

type ShapeGroup struct {
	Label       string       `json:"label"`
	Description string       `json:"description"`
	Shapes      []ShapeSpec `json:"shapes"`
}

type ShapeSpec struct {
	Name        string  `json:"name"`
	Processor   string  `json:"processor"`
	MinOCPU     float64 `json:"min_ocpu"`
	MaxOCPU     float64 `json:"max_ocpu"`
	MinMemory   float64 `json:"min_memory"`
	MaxMemory   float64 `json:"max_memory"`
	MaxNetwork  string  `json:"max_network"`
	Description string  `json:"description,omitempty"`
}

var allShapes = map[string]ShapeSpec{
	"VM.Standard.E4.Flex":  {Name: "VM.Standard.E4.Flex", Processor: "AMD EPYC 7J13 (Milan)", MinOCPU: 1, MaxOCPU: 64, MinMemory: 1, MaxMemory: 1024, MaxNetwork: "40 Gbps", Description: "2.55 GHz · up to 64 OCPU · recommended for VPS"},
	"VM.Standard.E5.Flex":  {Name: "VM.Standard.E5.Flex", Processor: "AMD EPYC 9J14 (Genoa)", MinOCPU: 1, MaxOCPU: 94, MinMemory: 1, MaxMemory: 1049, MaxNetwork: "40 Gbps", Description: "2.4 GHz · up to 94 OCPU · best price-performance"},
	"VM.Standard.E6.Flex":  {Name: "VM.Standard.E6.Flex", Processor: "AMD EPYC 9J45", MinOCPU: 1, MaxOCPU: 126, MinMemory: 1, MaxMemory: 1454, MaxNetwork: "99 Gbps", Description: "2.7 GHz · up to 126 OCPU · newest AMD"},
	"VM.Standard.E6.Ax.Flex": {Name: "VM.Standard.E6.Ax.Flex", Processor: "AMD EPYC 9J45", MinOCPU: 1, MaxOCPU: 94, MinMemory: 1, MaxMemory: 712, MaxNetwork: "99 Gbps", Description: "2.7 GHz · up to 94 OCPU · lower memory tier"},

	"VM.Standard3.Flex":    {Name: "VM.Standard3.Flex", Processor: "Intel Xeon Platinum 8358 (Ice Lake)", MinOCPU: 1, MaxOCPU: 32, MinMemory: 1, MaxMemory: 512, MaxNetwork: "32 Gbps", Description: "2.6 GHz · up to 32 OCPU · broadest compatibility"},
	"VM.Standard4.Ax.Flex": {Name: "VM.Standard4.Ax.Flex", Processor: "Intel Xeon 6987P-C (Xeon 6)", MinOCPU: 1, MaxOCPU: 39, MinMemory: 1, MaxMemory: 360, MaxNetwork: "99 Gbps", Description: "2.2 GHz · up to 39 OCPU · newest Intel"},

	"VM.Standard.A1.Flex":  {Name: "VM.Standard.A1.Flex", Processor: "Ampere Altra Q80-30", MinOCPU: 1, MaxOCPU: 76, MinMemory: 1, MaxMemory: 472, MaxNetwork: "40 Gbps", Description: "3.0 GHz · up to 76 OCPU · best price · Always Free eligible"},
	"VM.Standard.A2.Flex":  {Name: "VM.Standard.A2.Flex", Processor: "AmpereOne A160-30", MinOCPU: 1, MaxOCPU: 78, MinMemory: 1, MaxMemory: 946, MaxNetwork: "78 Gbps", Description: "3.0 GHz · up to 78 OCPU · high memory ARM"},
	"VM.Standard.A4.Flex":  {Name: "VM.Standard.A4.Flex", Processor: "AmpereOne M A06-36M", MinOCPU: 1, MaxOCPU: 45, MinMemory: 1, MaxMemory: 700, MaxNetwork: "100 Gbps", Description: "3.6 GHz · up to 45 OCPU · newest ARM"},
	"VM.Standard.A4.Ax.Flex": {Name: "VM.Standard.A4.Ax.Flex", Processor: "AmpereOne M 192-36M", MinOCPU: 1, MaxOCPU: 45, MinMemory: 1, MaxMemory: 720, MaxNetwork: "100 Gbps", Description: "3.6 GHz · up to 45 OCPU · higher memory tier"},
}

var shapeGroups = []ShapeGroup{
	{
		Label:       "AMD",
		Description: "AMD EPYC — best value for x86 workloads",
		Shapes:      []ShapeSpec{allShapes["VM.Standard.E4.Flex"], allShapes["VM.Standard.E5.Flex"], allShapes["VM.Standard.E6.Flex"], allShapes["VM.Standard.E6.Ax.Flex"]},
	},
	{
		Label:       "Intel",
		Description: "Intel Xeon — broadest software compatibility",
		Shapes:      []ShapeSpec{allShapes["VM.Standard3.Flex"], allShapes["VM.Standard4.Ax.Flex"]},
	},
	{
		Label:       "Ampere",
		Description: "Ampere ARM64 — best price-performance, lowest cost",
		Shapes:      []ShapeSpec{allShapes["VM.Standard.A1.Flex"], allShapes["VM.Standard.A2.Flex"], allShapes["VM.Standard.A4.Flex"], allShapes["VM.Standard.A4.Ax.Flex"]},
	},
}

const (
	MinBootVolumeGB = 10
	MaxBootVolumeGB = 200
)

func ValidateShape(shape string) error {
	if _, ok := allShapes[shape]; !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	return nil
}

func ValidateOCPU(shape string, ocpu float64) error {
	spec, ok := allShapes[shape]
	if !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	if ocpu < spec.MinOCPU || ocpu > spec.MaxOCPU {
		return fmt.Errorf("ocpu %.1f out of range [%.0f, %.0f] for shape %s", ocpu, spec.MinOCPU, spec.MaxOCPU, shape)
	}
	return nil
}

func ValidateMemory(shape string, memory float64) error {
	spec, ok := allShapes[shape]
	if !ok {
		return fmt.Errorf("unsupported shape: %s", shape)
	}
	if memory < spec.MinMemory || memory > spec.MaxMemory {
		return fmt.Errorf("memory %.1f GB out of range [%.0f, %.0f] for shape %s", memory, spec.MinMemory, spec.MaxMemory, shape)
	}
	return nil
}

func ValidateBootVolume(gb int) error {
	if gb < MinBootVolumeGB || gb > MaxBootVolumeGB {
		return fmt.Errorf("boot volume %d GB out of range [%d, %d]", gb, MinBootVolumeGB, MaxBootVolumeGB)
	}
	return nil
}

func ShapeGroups() []ShapeGroup {
	return shapeGroups
}

func SupportedShapes() []string {
	shapes := make([]string, 0, len(allShapes))
	for s := range allShapes {
		shapes = append(shapes, s)
	}
	return shapes
}
