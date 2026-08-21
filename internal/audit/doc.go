// Package audit defines types and methods that implement audit functionality.
// Audit service receives events from AuditRecorder (middleware package),
// enqueues them and provides transmission to subscribers (Observer).
package audit
