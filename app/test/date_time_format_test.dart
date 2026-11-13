import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/date_time.dart';

void main() {
  test('formatDateTime shows local date and time, empty for null', () {
    expect(formatDateTime(null), '');
    final local = DateTime(2026, 11, 12, 15, 5);
    expect(formatDateTime(local), '2026-11-12 15:05');
    expect(formatDateTime(DateTime(2026, 1, 2, 3, 4)), '2026-01-02 03:04');
  });
}
