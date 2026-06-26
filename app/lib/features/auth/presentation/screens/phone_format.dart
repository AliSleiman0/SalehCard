/// Converts a locally-entered Lebanese mobile number to E.164. The UI fixes the
/// country to +961 (see CountryCodeBox); this strips spaces and a single leading
/// zero and prepends the prefix. The backend re-validates, so this only needs to
/// produce a sane candidate.
String toE164Lebanon(String raw) {
  var digits = raw.replaceAll(RegExp(r'\D'), '');
  if (digits.startsWith('0')) digits = digits.substring(1);
  return '+961$digits';
}
