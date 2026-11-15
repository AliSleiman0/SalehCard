import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/order_number.dart';

void main() {
  test('orderNumber is # + the last 8 characters, upper-cased', () {
    expect(orderNumber('6af90254ce0bb4698d03458d'), '#8D03458D');
    expect(orderNumber('abc'), '#ABC');
  });
}
