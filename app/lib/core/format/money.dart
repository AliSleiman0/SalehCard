import 'package:intl/intl.dart';

/// Formats a USD amount. The system supports USD only, so the symbol is fixed;
/// the numeral formatting uses en_US (Western digits) regardless of UI locale.
String formatUsd(double amount) {
  return NumberFormat.currency(
    locale: 'en_US',
    symbol: r'$',
    decimalDigits: 2,
  ).format(amount);
}
