// Package forecast provides quota time-to-exhaustion (TTX) predictions and cost anomaly detection.
//
// Key types:
//   - GroupForecast: Represents exhaustion time, remaining fraction, and confidence for a single quota group
//   - AccountForecast: Combines all group forecasts for a given account
//   - Anomaly: Holds stats about a detected cost spike relative to historical spend averages
//   - AnomalyConfig: Configures rolling window and Z-score parameters for cost anomaly checks
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - anomaly.go: Implements statistical cost anomaly detection via rolling Z-scores
//   - forecast.go: Implements sliding-window burn rate calculations and TTX projections
package forecast
