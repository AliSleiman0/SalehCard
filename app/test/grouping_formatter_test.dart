import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/grouping_formatter.dart';

TextEditingValue _type(String text, [int? caret]) => TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: caret ?? text.length),
    );

void main() {
  const f = ThousandsGroupingFormatter();

  test('groupDigits groups in threes and drops leading zeros', () {
    expect(groupDigits(''), '');
    expect(groupDigits('0'), '0');
    expect(groupDigits('5'), '5');
    expect(groupDigits('1000'), '1,000');
    expect(groupDigits('45000'), '45,000');
    expect(groupDigits('1234567'), '1,234,567');
    expect(groupDigits('007'), '7');
  });

  test('formats as the customer types at the end', () {
    final out = f.formatEditUpdate(_type('100'), _type('1000'));
    expect(out.text, '1,000');
    expect(out.selection.baseOffset, 5);
  });

  test('re-groups after a delete and keeps the caret by the same digit', () {
    // "1,234,567" → caret after "4", user deletes the "4": "1,23,567" → caret 4
    final out = f.formatEditUpdate(_type('1,234,567', 5), _type('1,23,567', 4));
    expect(out.text, '123,567');
    expect(out.selection.baseOffset, 3);
  });

  test('stripGrouping undoes the separators for parsing', () {
    expect(stripGrouping('1,234,567'), '1234567');
    expect(double.parse(stripGrouping('5,000')), 5000);
  });
}
