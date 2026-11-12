import 'package:flutter/services.dart';

/// Inserts thousands separators into a digits-only field as the customer types
/// ("1000" → "1,000"), keeping the caret in the same place relative to the
/// digits. Put it AFTER a digits filter. Readers must strip the commas before
/// parsing — use [stripGrouping].
class ThousandsGroupingFormatter extends TextInputFormatter {
  const ThousandsGroupingFormatter();

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final digits = stripGrouping(newValue.text);
    final grouped = groupDigits(digits);
    if (grouped == newValue.text) return newValue;

    // Count the digits left of the caret, then find the same digit position in
    // the grouped string so editing mid-number doesn't jump to the end.
    final caret = newValue.selection.baseOffset.clamp(0, newValue.text.length);
    final digitsBeforeCaret =
        stripGrouping(newValue.text.substring(0, caret)).length;
    var offset = 0;
    var seen = 0;
    while (offset < grouped.length && seen < digitsBeforeCaret) {
      if (grouped[offset] != ',') seen++;
      offset++;
    }
    return TextEditingValue(
      text: grouped,
      selection: TextSelection.collapsed(offset: offset),
    );
  }
}

/// Removes the grouping commas a [ThousandsGroupingFormatter] inserted.
String stripGrouping(String s) => s.replaceAll(',', '');

/// Groups a run of ASCII digits in threes: "1234567" → "1,234,567". Leading
/// zeros are dropped (but a lone "0" stays).
String groupDigits(String digits) {
  final d = digits.replaceFirst(RegExp(r'^0+(?=\d)'), '');
  final buf = StringBuffer();
  for (var i = 0; i < d.length; i++) {
    if (i > 0 && (d.length - i) % 3 == 0) buf.write(',');
    buf.write(d[i]);
  }
  return buf.toString();
}
