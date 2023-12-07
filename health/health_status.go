//go:generate go-enum -f=$GOFILE --marshal

package health

// HealthStatus defines the health statuses.
/* ENUM(
NotSet
OK
Warning
Critical
)
*/
type HealthStatus int
