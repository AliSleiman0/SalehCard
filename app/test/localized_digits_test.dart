import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/localized_digits.dart';

TextEditingValue _type(List<TextInputFormatter> fs, String text) {
  var value = TextEditingValue(
    text: text,
    selection: TextSelection.collapsed(offset: text.length),
  );
  for (final f in fs) {
    value = f.formatEditUpdate(TextEditingValue.empty, value);
  }
  return value;
}

void main() {
  group('normalizeDigits', () {
    test('maps Arabic-Indic and Persian digits to 0-9', () {
      expect(normalizeDigits('٠١٢٣٤٥٦٧٨٩'), '0123456789');
      expect(normalizeDigits('۰۱۲۳۴۵۶۷۸۹'), '0123456789');
      expect(normalizeDigits('٤٥٠٠٠'), '45000');
    });

    test('leaves other characters untouched', () {
      expect(normalizeDigits('abc 12.5'), 'abc 12.5');
      expect(normalizeDigits(''), '');
      expect(normalizeDigits('x٣y'), 'x3y');
    });
  });

  group('LocalizedDigitsFormatter', () {
    test('runs before digitsOnly so Arabic digits survive', () {
      final withFix = _type([
        const LocalizedDigitsFormatter(),
        FilteringTextInputFormatter.digitsOnly,
      ], '٤٥٠٠٠');
      expect(withFix.text, '45000');
      expect(withFix.selection.baseOffset, 5);
      // Without it, the digits filter strips every Arabic digit.
      final without =
          _type([FilteringTextInputFormatter.digitsOnly], '٤٥٠٠٠');
      expect(without.text, '');
    });

    test('works with the decimal amount filter', () {
      final v = _type([
        const LocalizedDigitsFormatter(),
        FilteringTextInputFormatter.allow(RegExp(r'[0-9.]')),
      ], '۱۲.۵');
      expect(v.text, '12.5');
    });

    test('returns the value unchanged when there is nothing to map', () {
      const v = TextEditingValue(text: '123');
      expect(
        const LocalizedDigitsFormatter()
            .formatEditUpdate(TextEditingValue.empty, v),
        same(v),
      );
    });
  });
}
