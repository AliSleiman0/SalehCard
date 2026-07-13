import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/catalog/domain/entities/product.dart';
import 'package:salehcard_app/features/checkout/domain/field_validation.dart';

void main() {
  const bounded = InputFieldConstraints(min: 1, max: 100);

  group('checkAmountBounds', () {
    test('null constraints are unbounded', () {
      expect(checkAmountBounds(null, '999999'), AmountBoundsIssue.none);
      expect(checkAmountBounds(null, 'abc'), AmountBoundsIssue.none);
    });

    test('no bounds set is unbounded', () {
      const c = InputFieldConstraints(options: ['x']);
      expect(checkAmountBounds(c, '999999'), AmountBoundsIssue.none);
    });

    test('legacy corrupt {0,0} is unbounded', () {
      const c = InputFieldConstraints(min: 0, max: 0);
      expect(checkAmountBounds(c, '999999'), AmountBoundsIssue.none);
    });

    test('inverted min>max is unbounded', () {
      const c = InputFieldConstraints(min: 500, max: 100);
      expect(checkAmountBounds(c, '1'), AmountBoundsIssue.none);
    });

    test('empty value is not a bounds problem', () {
      expect(checkAmountBounds(bounded, ''), AmountBoundsIssue.none);
      expect(checkAmountBounds(bounded, '   '), AmountBoundsIssue.none);
    });

    test('non-numeric value with active bounds is flagged', () {
      expect(checkAmountBounds(bounded, 'abc'), AmountBoundsIssue.notANumber);
      expect(checkAmountBounds(bounded, '1.2.3'), AmountBoundsIssue.notANumber);
    });

    test('boundaries are inclusive', () {
      expect(checkAmountBounds(bounded, '1'), AmountBoundsIssue.none);
      expect(checkAmountBounds(bounded, '100'), AmountBoundsIssue.none);
    });

    test('out-of-range values are flagged', () {
      expect(checkAmountBounds(bounded, '0.5'), AmountBoundsIssue.belowMin);
      expect(checkAmountBounds(bounded, '500'), AmountBoundsIssue.aboveMax);
    });

    test('one-sided bounds enforce independently', () {
      const minOnly = InputFieldConstraints(min: 5);
      const maxOnly = InputFieldConstraints(max: 10);
      expect(checkAmountBounds(minOnly, '4'), AmountBoundsIssue.belowMin);
      expect(checkAmountBounds(minOnly, '999999'), AmountBoundsIssue.none);
      expect(checkAmountBounds(maxOnly, '11'), AmountBoundsIssue.aboveMax);
      expect(checkAmountBounds(maxOnly, '0'), AmountBoundsIssue.none);
    });

    test('legitimate 0 min with non-zero max is a real bound', () {
      const c = InputFieldConstraints(min: 0, max: 10);
      expect(checkAmountBounds(c, '5'), AmountBoundsIssue.none);
      expect(checkAmountBounds(c, '11'), AmountBoundsIssue.aboveMax);
    });
  });

  group('hasActiveBounds', () {
    test('mirrors the enforcement rule', () {
      expect(hasActiveBounds(null), false);
      expect(hasActiveBounds(const InputFieldConstraints(options: ['x'])), false);
      expect(hasActiveBounds(const InputFieldConstraints(min: 0, max: 0)), false);
      expect(hasActiveBounds(const InputFieldConstraints(min: 9, max: 1)), false);
      expect(hasActiveBounds(bounded), true);
      expect(hasActiveBounds(const InputFieldConstraints(min: 1)), true);
      expect(hasActiveBounds(const InputFieldConstraints(max: 1)), true);
    });
  });

  group('formatBound', () {
    test('drops a trailing .0 and keeps real decimals', () {
      expect(formatBound(5.0), '5');
      expect(formatBound(5000.0), '5000');
      expect(formatBound(0.5), '0.5');
    });
  });
}
