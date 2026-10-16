import 'package:flutter/services.dart';

/// Maps Arabic-Indic (٠-٩, U+0660–0669) and Extended Arabic-Indic / Persian
/// (۰-۹, U+06F0–06F9) digits in [input] to their Western 0-9 counterparts;
/// every other character is left untouched.
String normalizeDigits(String input) {
  final buf = StringBuffer();
  for (final rune in input.runes) {
    if (rune >= 0x0660 && rune <= 0x0669) {
      buf.writeCharCode(0x30 + rune - 0x0660);
    } else if (rune >= 0x06F0 && rune <= 0x06F9) {
      buf.writeCharCode(0x30 + rune - 0x06F0);
    } else {
      buf.writeCharCode(rune);
    }
  }
  return buf.toString();
}

/// Rewrites Arabic-Indic / Persian digits to Western 0-9 as they're typed.
/// Put it BEFORE a digits filter (e.g. [FilteringTextInputFormatter.digitsOnly])
/// — otherwise an Arabic keyboard's ٠-٩ are stripped and nothing gets typed.
/// Each mapped digit is still one UTF-16 code unit, so the selection offsets
/// carry over unchanged.
class LocalizedDigitsFormatter extends TextInputFormatter {
  const LocalizedDigitsFormatter();

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final text = normalizeDigits(newValue.text);
    if (text == newValue.text) return newValue;
    return newValue.copyWith(text: text);
  }
}
