import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../domain/password_rules.dart';
import '../providers.dart';

/// Self-service password change: current + new + confirm, validated on the
/// device (min 8 characters, max 72 bytes, confirmation matches) and
/// re-checked by the API. Other sessions stay signed in, so the copy never
/// promises "signed out everywhere".
class ChangePasswordScreen extends ConsumerStatefulWidget {
  const ChangePasswordScreen({super.key});

  @override
  ConsumerState<ChangePasswordScreen> createState() =>
      _ChangePasswordScreenState();
}

class _ChangePasswordScreenState extends ConsumerState<ChangePasswordScreen> {
  final _current = TextEditingController();
  final _next = TextEditingController();
  final _confirm = TextEditingController();
  String? _error;
  bool _noPassword = false;

  @override
  void dispose() {
    _current.dispose();
    _next.dispose();
    _confirm.dispose();
    super.dispose();
  }

  String _message(PasswordProblem p, AppLocalizations l10n) => switch (p) {
    PasswordProblem.currentRequired => l10n.changePasswordErrCurrentRequired,
    PasswordProblem.tooShort => l10n.changePasswordErrShort,
    PasswordProblem.tooLong => l10n.changePasswordErrLong,
    PasswordProblem.mismatch => l10n.changePasswordErrMismatch,
    PasswordProblem.wrongCurrent => l10n.changePasswordErrWrong,
    PasswordProblem.noPasswordSet => l10n.changePasswordErrNoPassword,
    PasswordProblem.rateLimited => l10n.changePasswordErrRateLimited,
  };

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final problem = validateChangePassword(
      current: _current.text,
      next: _next.text,
      confirm: _confirm.text,
    );
    if (problem != null) {
      setState(() {
        _error = _message(problem, l10n);
        _noPassword = false;
      });
      return;
    }
    FocusScope.of(context).unfocus();
    setState(() {
      _error = null;
      _noPassword = false;
    });
    final ok = await ref
        .read(changePasswordControllerProvider.notifier)
        .submit(currentPassword: _current.text, newPassword: _next.text);
    if (!mounted) return;
    if (ok) {
      _current.clear();
      _next.clear();
      _confirm.clear();
      messenger.showSnackBar(SnackBar(content: Text(l10n.changePasswordDone)));
      return;
    }
    final failure = ref.read(changePasswordControllerProvider).failure;
    final mapped = failure is ServerFailure
        ? passwordProblemForCode(failure.code)
        : null;
    setState(() {
      _noPassword = mapped == PasswordProblem.noPasswordSet;
      _error = mapped != null
          ? _message(mapped, l10n)
          : ((failure?.message.isNotEmpty ?? false)
                ? failure!.message
                : l10n.changePasswordErrGeneric);
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(changePasswordControllerProvider).submitting;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.changePasswordTitle),
      ),
      body: AutofillGroup(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
          children: [
            Text(
              l10n.changePasswordIntro,
              style: TextStyle(color: colors.textDim),
            ),
            const SizedBox(height: 8),
            _PasswordField(
              key: const Key('pw-current'),
              label: l10n.changePasswordCurrent,
              controller: _current,
              autofillHints: const [AutofillHints.password],
              textInputAction: TextInputAction.next,
            ),
            _PasswordField(
              key: const Key('pw-new'),
              label: l10n.changePasswordNew,
              controller: _next,
              helper: l10n.changePasswordRules,
              autofillHints: const [AutofillHints.newPassword],
              textInputAction: TextInputAction.next,
            ),
            _PasswordField(
              key: const Key('pw-confirm'),
              label: l10n.changePasswordConfirm,
              controller: _confirm,
              autofillHints: const [AutofillHints.newPassword],
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => submitting ? null : _submit(),
            ),
            if (_error != null) ...[
              const SizedBox(height: 14),
              Text(
                _error!,
                key: const Key('pw-error'),
                style: const TextStyle(
                  color: AppTokens.danger,
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (_noPassword) ...[
                const SizedBox(height: 6),
                Text(
                  l10n.changePasswordNoPasswordHint,
                  style: TextStyle(color: colors.textDim),
                ),
              ],
            ],
            const SizedBox(height: 22),
            SizedBox(
              height: 52,
              child: FilledButton(
                key: const Key('pw-submit'),
                onPressed: submitting ? null : _submit,
                child: submitting
                    ? const SizedBox(
                        width: 22,
                        height: 22,
                        child: CircularProgressIndicator(strokeWidth: 2.4),
                      )
                    : Text(l10n.changePasswordSubmit),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A labelled password field with a show/hide toggle (the suffix sits on the
/// trailing edge, so it flips with RTL).
class _PasswordField extends StatefulWidget {
  const _PasswordField({
    super.key,
    required this.label,
    required this.controller,
    this.helper,
    this.autofillHints,
    this.textInputAction,
    this.onSubmitted,
  });

  final String label;
  final TextEditingController controller;
  final String? helper;
  final Iterable<String>? autofillHints;
  final TextInputAction? textInputAction;
  final ValueChanged<String>? onSubmitted;

  @override
  State<_PasswordField> createState() => _PasswordFieldState();
}

class _PasswordFieldState extends State<_PasswordField> {
  bool _shown = false;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    return Padding(
      padding: const EdgeInsets.only(top: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              widget.label,
              style: TextStyle(fontWeight: FontWeight.w700, color: colors.text),
            ),
          ),
          TextField(
            controller: widget.controller,
            obscureText: !_shown,
            enableSuggestions: false,
            autocorrect: false,
            autofillHints: widget.autofillHints,
            textInputAction: widget.textInputAction,
            onSubmitted: widget.onSubmitted,
            decoration: InputDecoration(
              helperText: widget.helper,
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(AppTokens.rMd),
              ),
              suffixIcon: IconButton(
                tooltip: _shown
                    ? l10n.changePasswordHide
                    : l10n.changePasswordShow,
                icon: Icon(
                  _shown
                      ? Icons.visibility_off_outlined
                      : Icons.visibility_outlined,
                ),
                onPressed: () => setState(() => _shown = !_shown),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
