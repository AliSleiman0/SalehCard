import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/auth_header.dart';
import '../../../../core/widgets/auth_switch_link.dart';
import '../../../../core/widgets/auth_text_field.dart';
import '../../../../core/widgets/country_code_box.dart';
import '../../../../core/widgets/gradient_heading.dart';
import '../../../../core/widgets/otp_input.dart';
import '../../../../core/widgets/primary_cta.dart';
import '../controllers/auth_controller.dart';
import 'phone_format.dart';

/// Sign-in offering both real auth paths: a passwordless **code** (phone → OTP)
/// and **password** (phone + password). The router redirects to /home on success.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

enum _Mode { code, password }

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _phone = TextEditingController();
  final _password = TextEditingController();
  String _otp = '';

  _Mode _mode = _Mode.code;
  bool _codeSent = false;
  String? _errorField; // 'phone' | 'password' | 'otp'
  String? _errorMessage;
  bool _loading = false;

  @override
  void dispose() {
    _phone.dispose();
    _password.dispose();
    super.dispose();
  }

  int get _digits => _phone.text.replaceAll(RegExp(r'\D'), '').length;

  void _clearError() {
    if (_errorField != null) setState(() => _errorField = _errorMessage = null);
  }

  void _setError(String field, String message) {
    setState(() {
      _errorField = field;
      _errorMessage = message;
    });
  }

  void _snack(String message) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
  }

  void _switchMode(_Mode mode) {
    setState(() {
      _mode = mode;
      _codeSent = false;
      _errorField = _errorMessage = null;
    });
  }

  Future<void> _submit() async {
    if (_loading) return;
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(authControllerProvider.notifier);

    // CODE mode, step 2: verify the OTP (phone already validated when sent).
    if (_mode == _Mode.code && _codeSent) {
      if (_otp.length < 6) return _setError('otp', l10n.otpIncomplete);
      setState(() {
        _errorField = _errorMessage = null;
        _loading = true;
      });
      final failure = await notifier.verifyOtp(
        phone: toE164Lebanon(_phone.text),
        code: _otp,
      );
      if (!mounted) return;
      setState(() => _loading = false);
      if (failure != null) _setError('otp', failure.message);
      return;
    }

    if (_digits < 7) return _setError('phone', l10n.invalidPhone);
    final phone = toE164Lebanon(_phone.text);

    // PASSWORD mode: phone + password → login.
    if (_mode == _Mode.password) {
      if (_password.text.isEmpty) {
        return _setError('password', l10n.passwordRequired);
      }
      setState(() {
        _errorField = _errorMessage = null;
        _loading = true;
      });
      final failure =
          await notifier.signInWithPhone(phone: phone, password: _password.text);
      if (!mounted) return;
      setState(() => _loading = false);
      if (failure != null) _snack(failure.message);
      return;
    }

    // CODE mode, step 1: request the OTP.
    setState(() {
      _errorField = _errorMessage = null;
      _loading = true;
    });
    final failure = await notifier.requestOtp(phone);
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (failure == null) _codeSent = true;
    });
    if (failure != null) _snack(failure.message);
  }

  Future<void> _resend() async {
    final failure =
        await ref.read(authControllerProvider.notifier).requestOtp(toE164Lebanon(_phone.text));
    if (!mounted) return;
    _snack(failure?.message ?? AppLocalizations.of(context).sendCodeButton);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final verifying = _mode == _Mode.code && _codeSent;

    final String cta;
    if (_mode == _Mode.password) {
      cta = l10n.signInButton;
    } else {
      cta = _codeSent ? l10n.verifyButton : l10n.sendCodeButton;
    }

    return Scaffold(
      backgroundColor: colors.surface,
      body: SafeArea(
        child: Column(
          children: [
            AuthHeader(
              onBack:
                  verifying ? () => setState(() => _codeSent = false) : null,
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GradientHeading(l10n.welcomeBackTitle),
                    const SizedBox(height: 10),
                    Text(
                      verifying
                          ? l10n.otpSubtitle('+961 ${_phone.text}')
                          : l10n.signInSubtitle,
                      style: TextStyle(
                        fontSize: 15.5,
                        height: 1.5,
                        color: colors.textDim,
                      ),
                    ),
                    const SizedBox(height: 28),
                    if (verifying)
                      ..._otpStep(l10n, colors)
                    else ...[
                      AuthTextField(
                        label: l10n.mobileNumberLabel,
                        controller: _phone,
                        hintText: l10n.phoneHint,
                        keyboardType: TextInputType.phone,
                        autofillHints: const [AutofillHints.telephoneNumber],
                        inputFormatters: [
                          FilteringTextInputFormatter.allow(RegExp(r'[0-9 ]')),
                          LengthLimitingTextInputFormatter(12),
                        ],
                        leading: const CountryCodeBox(),
                        errorText:
                            _errorField == 'phone' ? _errorMessage : null,
                        onChanged: (_) => _clearError(),
                      ),
                      if (_mode == _Mode.password) ...[
                        const SizedBox(height: 20),
                        AuthTextField(
                          label: l10n.passwordLabel,
                          controller: _password,
                          hintText: l10n.passwordHint,
                          obscureText: true,
                          textInputAction: TextInputAction.done,
                          errorText:
                              _errorField == 'password' ? _errorMessage : null,
                          onChanged: (_) => _clearError(),
                          onSubmitted: (_) => _submit(),
                        ),
                      ],
                    ],
                    const SizedBox(height: 30),
                    PrimaryCta(label: cta, onPressed: _submit, loading: _loading),
                    const SizedBox(height: 12),
                    if (!verifying)
                      Center(
                        child: TextButton(
                          onPressed: () => _switchMode(
                            _mode == _Mode.code ? _Mode.password : _Mode.code,
                          ),
                          child: Text(
                            _mode == _Mode.code
                                ? l10n.usePasswordInstead
                                : l10n.useCodeInstead,
                          ),
                        ),
                      ),
                    const SizedBox(height: 6),
                    AuthSwitchLink(
                      prefix: l10n.noAccountPrefix,
                      link: l10n.signUpButton,
                      onTap: () => context.go('/signup'),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _otpStep(AppLocalizations l10n, AppColors colors) {
    return [
      OtpInput(
        hasError: _errorField == 'otp',
        onChanged: (v) {
          _otp = v;
          _clearError();
        },
      ),
      if (_errorField == 'otp') ...[
        const SizedBox(height: 9),
        Text(
          _errorMessage!,
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w500,
            color: AppTokens.danger,
          ),
        ),
      ],
      const SizedBox(height: 16),
      Row(
        children: [
          Text(
            l10n.resendPrefix,
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w500,
              color: colors.textDim,
            ),
          ),
          GestureDetector(
            onTap: _resend,
            child: Text(
              l10n.resendLink,
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: AppTokens.brand1,
              ),
            ),
          ),
        ],
      ),
    ];
  }
}
