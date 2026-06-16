// Package ir defines sysgo's Domain-Driven intermediate representation. The IR
// is decoupled from the SysML metamodel; templates render only against it.
package ir

// Project is the root of the IR — a Go module plus its bounded contexts.
type Project struct {
	Module   string
	Contexts []*Context
}

// Context is a bounded context (a SysML Package), mapped to a top-level Go
// module subtree.
type Context struct {
	Name    string
	Package string
	Dir     string

	Entities       []*Entity
	ValueObjects   []*ValueObject
	DomainServices []*DomainService
	UseCases       []*UseCase
	DrivenPorts    []*Port // repository/gateway interfaces (port/out)
	DrivingPorts   []*Port // use-case boundary interfaces (port/in)
	Events         []*DomainEvent
}

// Entity is a domain entity; Aggregate marks an aggregate root.
type Entity struct {
	Name      string
	Aggregate bool
	Fields    []*Field
	Methods   []*Method
	Meta      Metadata
}

// ValueObject is an immutable, identity-less domain value.
type ValueObject struct {
	Name   string
	Fields []*Field
	Meta   Metadata
}

// DomainService is a stateless domain operation (interface + scaffold impl).
type DomainService struct {
	Name    string
	Methods []*Method
	Meta    Metadata
}

// DomainEvent is a fact that happened in the domain.
type DomainEvent struct {
	Name   string
	Fields []*Field
	Meta   Metadata
}

// Field is a struct field with its resolved Go type.
type Field struct {
	Name     string
	GoType   string
	Optional bool
	Pointer  bool
	Tags     string
	Doc      string
}

// Method is an interface method / behavioral signature.
type Method struct {
	Name    string
	Params  []*Param
	Results []*Param
	Doc     string
}

// Param is a method parameter or result.
type Param struct {
	Name   string
	GoType string
}

// PortDir is the direction of a port relative to the application core.
type PortDir int

const (
	// DirIn is a driving (primary/inbound) port.
	DirIn PortDir = iota
	// DirOut is a driven (secondary/outbound) port.
	DirOut
)

// PortKind classifies the role of a port interface.
type PortKind int

const (
	// KindUseCase is a driving use-case boundary interface.
	KindUseCase PortKind = iota
	// KindRepository is a driven persistence interface.
	KindRepository
	// KindGateway is a driven external-system interface.
	KindGateway
	// KindService is a driven service interface.
	KindService
)

// Port is a Go interface at an architectural boundary.
type Port struct {
	Name      string
	Direction PortDir
	Kind      PortKind
	Methods   []*Method
	Meta      Metadata
}

// DTO is a data structure crossing a port boundary.
type DTO struct {
	Name   string
	Fields []*Field
}

// UseCase is an application interactor / application service.
type UseCase struct {
	Name   string
	Input  *DTO
	Output *DTO
	Port   *Port
	Meta   Metadata
}

// Metadata carries resolved generation hints, merged from heuristics, in-model
// SysML metadata, overlay-injected keys, and config rules.
type Metadata struct {
	GoType              string
	GoName              string
	Tags                string
	SkipOptionalPointer bool
	Stereotype          string
	TargetDir           string
	TargetLayer         string
	Imports             []string
}
