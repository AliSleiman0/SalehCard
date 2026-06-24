import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Holds the active UI locale and toggles between English (LTR) and Arabic
/// (RTL). Flutter flips layout direction automatically from this locale.
class LocaleController extends Notifier<Locale> {
  @override
  Locale build() => const Locale('en');

  void toggle() {
    state = state.languageCode == 'en'
        ? const Locale('ar')
        : const Locale('en');
  }
}

final localeControllerProvider =
    NotifierProvider<LocaleController, Locale>(LocaleController.new);
