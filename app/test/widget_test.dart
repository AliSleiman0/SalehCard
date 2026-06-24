import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/i18n/i18n_string.dart';

void main() {
  group('I18nString.resolve', () {
    test('returns the requested locale when present', () {
      const s = I18nString({'en': 'Card', 'ar': 'بطاقة', 'tr': 'Kart'});
      expect(s.resolve('ar'), 'بطاقة');
      expect(s.resolve('en'), 'Card');
    });

    test('falls back to English when the locale is missing', () {
      const s = I18nString({'en': 'Card', 'tr': 'Kart'});
      expect(s.resolve('ar'), 'Card');
    });

    test('falls back to any value when English is missing', () {
      const s = I18nString({'tr': 'Kart'});
      expect(s.resolve('ar'), 'Kart');
    });

    test('returns empty string for an empty map', () {
      const s = I18nString({});
      expect(s.resolve('en'), '');
    });

    test('fromJson wraps a bare string as English', () {
      final s = I18nString.fromJson('Card');
      expect(s.resolve('ar'), 'Card');
    });
  });
}
