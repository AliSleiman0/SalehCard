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

/// Formats a USDT deposit amount digit-exact, no symbol. Shared-address
/// payments are identified by sub-cent "salt" digits, so the customer must
/// send exactly this value — rounding to 2 decimals would strip the digits
/// that matter. Up to 6 decimals (the TRC20 base unit), trailing zeros
/// trimmed, never fewer than 2 decimals: 10.0 → "10.00", 10.1 → "10.10",
/// 10.0037 → "10.0037".
String formatUsdtAmount(double amount) {
  var s = amount.toStringAsFixed(6);
  final dot = s.indexOf('.');
  while (s.length - dot - 1 > 2 && s.endsWith('0')) {
    s = s.substring(0, s.length - 1);
  }
  return s;
}

/// Formats a unit count (coins, points…) with thousands grouping in Western
/// digits, matching the server's denomination snapshot: 45000 → "45,000",
/// 1500.5 → "1,500.5".
String formatUnits(double units) {
  return NumberFormat('#,##0.##', 'en_US').format(units);
}
