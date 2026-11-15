import 'dart:convert';

/// Client-side rules for the self-service password change, mirroring the API
/// (`api/internal/modules/user/password.go`): at least 8 characters, at most
/// 72 UTF-8 BYTES (bcrypt's limit — an Arabic letter is 2 bytes), and the
/// confirmation must match. The server re-checks everything.
const minPasswordChars = 8;
const maxPasswordBytes = 72;

enum PasswordProblem {
  currentRequired,
  tooShort,
  tooLong,
  mismatch,
  wrongCurrent,
  noPasswordSet,
  rateLimited,
}

int utf8ByteLength(String s) => utf8.encode(s).length;

/// The first problem with the form, or null when it may be submitted.
PasswordProblem? validateChangePassword({
  required String current,
  required String next,
  required String confirm,
}) {
  if (current.isEmpty) return PasswordProblem.currentRequired;
  if (next.runes.length < minPasswordChars) return PasswordProblem.tooShort;
  if (utf8ByteLength(next) > maxPasswordBytes) return PasswordProblem.tooLong;
  if (next != confirm) return PasswordProblem.mismatch;
  return null;
}

/// Maps an API error code to its problem; null for unknown codes (the caller
/// shows the server's own message).
PasswordProblem? passwordProblemForCode(String code) => switch (code) {
  'WRONG_PASSWORD' => PasswordProblem.wrongCurrent,
  'WEAK_PASSWORD' => PasswordProblem.tooShort,
  'PASSWORD_TOO_LONG' => PasswordProblem.tooLong,
  'NO_PASSWORD_SET' => PasswordProblem.noPasswordSet,
  'RATE_LIMITED' => PasswordProblem.rateLimited,
  _ => null,
};
