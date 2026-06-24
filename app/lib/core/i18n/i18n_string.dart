/// A localized string from the backend, shaped `{en, ar, tr}`. Resolves to the
/// requested locale, falling back to English and then to any present value.
class I18nString {
  const I18nString(this.values);

  final Map<String, String> values;

  factory I18nString.fromJson(dynamic json) {
    if (json is Map) {
      return I18nString(
        json.map((key, value) =>
            MapEntry(key.toString(), value?.toString() ?? '')),
      );
    }
    if (json is String) {
      return I18nString({'en': json});
    }
    return const I18nString({});
  }

  String resolve(String localeCode) {
    final value = values[localeCode];
    if (value != null && value.isNotEmpty) return value;
    final english = values['en'];
    if (english != null && english.isNotEmpty) return english;
    return values.values.isNotEmpty ? values.values.first : '';
  }
}
