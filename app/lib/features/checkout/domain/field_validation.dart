import '../../catalog/domain/entities/product.dart';

/// Outcome of checking an `amount`-type field's value against its constraints.
enum AmountBoundsIssue { none, notANumber, belowMin, aboveMax }

/// Checks a raw `amount` input against the field's min/max bounds, mirroring
/// the server's tolerances exactly: no constraints, a `{min:0,max:0}` pair
/// (the known legacy-import corruption shape), or an inverted `min>max` pair
/// are all treated as unbounded. An empty value is [AmountBoundsIssue.none] —
/// emptiness is the required-check's job, not a bounds problem. Boundaries are
/// inclusive; one-sided bounds are enforced independently.
AmountBoundsIssue checkAmountBounds(InputFieldConstraints? c, String raw) {
  final min = c?.min;
  final max = c?.max;
  if (min == null && max == null) return AmountBoundsIssue.none;
  if (min != null && max != null && ((min == 0 && max == 0) || min > max)) {
    return AmountBoundsIssue.none; // corrupt legacy shapes — not real bounds
  }
  final value = raw.trim();
  if (value.isEmpty) return AmountBoundsIssue.none;
  final n = double.tryParse(value);
  if (n == null) return AmountBoundsIssue.notANumber;
  if (min != null && n < min) return AmountBoundsIssue.belowMin;
  if (max != null && n > max) return AmountBoundsIssue.aboveMax;
  return AmountBoundsIssue.none;
}

/// Whether the field carries bounds that [checkAmountBounds] would actually
/// enforce — drives the range hint under the input.
bool hasActiveBounds(InputFieldConstraints? c) {
  final min = c?.min;
  final max = c?.max;
  if (min == null && max == null) return false;
  if (min != null && max != null && ((min == 0 && max == 0) || min > max)) {
    return false;
  }
  return true;
}

/// Formats a bound for display, dropping a trailing `.0` (5.0 → "5").
String formatBound(double n) {
  return n == n.roundToDouble() ? n.round().toString() : n.toString();
}
