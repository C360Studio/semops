//go:build ignore
// +build ignore

// Package vocabulary provides controlled vocabulary constants for the robotics domain.
//
// This package defines entity types, relationship types, confidence levels, and
// resolution strategies used throughout the robotics processing pipeline. It serves
// as a single source of truth for domain-specific vocabulary, preventing typos
// and enabling compile-time checking of string literals.
//
// # Entity Types
//
// Entity types follow the domain:type format (e.g., "robotics:Drone") and are
// defined in entities.go. These constants should be used whenever creating or
// referencing robotics entities in the system.
//
// # Relationship Types
//
// Relationship constants are provided in relationships.go, which re-exports
// commonly used relationships from the global graph package and defines
// robotics-specific relationships like COMMUNICATES_WITH.
//
// # Confidence Levels
//
// Confidence constants in confidence.go provide standardized confidence values
// for different types of measurements and data sources, particularly for GPS
// positioning and battery monitoring.
//
// # Resolution Strategies
//
// Resolution strategy constants in resolution.go define how conflicts between
// duplicate entities should be resolved, supporting automated and manual
// resolution approaches.
//
// # Usage Example
//
//	import "github.com/c360/semops/pkg/processors/mavlink/vocabulary"
//
//	entity := &graph.Entity{
//		Type: vocabulary.EntityTypeDrone,
//		Confidence: vocabulary.GPSConfidence3DFix,
//	}
//
//	relationship := &graph.Relationship{
//		Type: vocabulary.RelTypePoweredBy,
//	}
package vocabulary
