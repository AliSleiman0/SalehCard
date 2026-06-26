import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../auth/domain/entities/user.dart';
import '../providers.dart';

/// Profile screen (`/profile`). Watches [profileProvider] (`GET /users/me`).
///
/// Two states toggled by the AppBar action:
///  - **view**: read-only account header (avatar + email), info rows, and the
///    saved player IDs as plain rows (or an [EmptyState] when none).
///  - **editing**: add a player ID (field + button) and remove existing ones
///    (chips with a delete affordance), then a sticky "Save changes" CTA that
///    `PATCH`es `savedPlayerIds`. Email is read-only (the backend has no name).
class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key});

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  final _inputController = TextEditingController();
  bool _editing = false;

  /// Working copy of the saved IDs while editing (seeded on enter-edit).
  List<String> _draftIds = const [];

  @override
  void dispose() {
    _inputController.dispose();
    super.dispose();
  }

  void _enterEdit(User user) {
    setState(() {
      _editing = true;
      _draftIds = List<String>.from(user.savedPlayerIds);
    });
    ref.read(profileControllerProvider.notifier).reset();
  }

  void _cancelEdit() {
    _inputController.clear();
    setState(() => _editing = false);
    ref.read(profileControllerProvider.notifier).reset();
  }

  void _addId() {
    final value = _inputController.text.trim();
    if (value.isEmpty || _draftIds.contains(value)) {
      _inputController.clear();
      return;
    }
    setState(() {
      _draftIds = [..._draftIds, value];
      _inputController.clear();
    });
  }

  void _removeId(String id) {
    setState(() => _draftIds = _draftIds.where((e) => e != id).toList());
  }

  Future<void> _save() async {
    final l10n = AppLocalizations.of(context);
    FocusScope.of(context).unfocus();
    final ok =
        await ref.read(profileControllerProvider.notifier).save(_draftIds);
    if (!mounted) return;
    if (ok) {
      setState(() => _editing = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.profileSaved)),
      );
    } else {
      final failure = ref.read(profileControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.loadFailed),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final profileAsync = ref.watch(profileProvider);
    final submitting = ref.watch(profileControllerProvider).submitting;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.profileTitle),
        actions: [
          profileAsync.maybeWhen(
            data: (user) => _editing
                ? TextButton(
                    onPressed: submitting ? null : _cancelEdit,
                    child: Text(l10n.cancel),
                  )
                : IconButton(
                    icon: const Icon(Icons.edit_outlined),
                    tooltip: l10n.editProfile,
                    onPressed: () => _enterEdit(user),
                  ),
            orElse: () => const SizedBox.shrink(),
          ),
        ],
      ),
      body: profileAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.loadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => ref.invalidate(profileProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (user) => Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsetsDirectional.fromSTEB(20, 18, 20, 24),
                children: [
                  _AccountHeader(email: user.email),
                  const SizedBox(height: 22),
                  _SectionLabel(l10n.accountInfo),
                  const SizedBox(height: 10),
                  _InfoCard(
                    rows: [
                      _InfoRow(label: l10n.emailLabel, value: user.email),
                      _InfoRow(label: l10n.roleLabel, value: user.role),
                    ],
                  ),
                  const SizedBox(height: 24),
                  _SectionLabel(l10n.savedPlayerIds),
                  const SizedBox(height: 10),
                  if (_editing)
                    _PlayerIdEditor(
                      controller: _inputController,
                      ids: _draftIds,
                      onAdd: _addId,
                      onRemove: _removeId,
                    )
                  else if (user.savedPlayerIds.isEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: EmptyState(
                        icon: Icons.badge_outlined,
                        title: l10n.savedPlayerIdsEmptyTitle,
                        message: l10n.savedPlayerIdsEmptySub,
                        actionLabel: l10n.editProfile,
                        onAction: () => _enterEdit(user),
                      ),
                    )
                  else
                    _InfoCard(
                      rows: [
                        for (final id in user.savedPlayerIds)
                          _InfoRow(
                            label: '',
                            value: id,
                            leading: Icons.badge_outlined,
                          ),
                      ],
                    ),
                ],
              ),
            ),
            if (_editing)
              _SaveBar(
                label: l10n.saveChanges,
                submitting: submitting,
                colors: colors,
                onTap: _save,
              ),
          ],
        ),
      ),
    );
  }
}

class _AccountHeader extends StatelessWidget {
  const _AccountHeader({required this.email});

  final String email;

  String get _initials {
    final trimmed = email.trim();
    if (trimmed.isEmpty) return '?';
    return trimmed.substring(0, 1).toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Row(
      children: [
        Container(
          width: 64,
          height: 64,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            gradient: AppTokens.brandGradient,
            shape: BoxShape.circle,
          ),
          child: Text(
            _initials,
            style: const TextStyle(
                color: Colors.white, fontSize: 26, fontWeight: FontWeight.w800),
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Text(
            email,
            style: TextStyle(
                fontSize: 16, fontWeight: FontWeight.w800, color: colors.text),
          ),
        ),
      ],
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: TextStyle(
          fontSize: 15,
          fontWeight: FontWeight.w800,
          color: context.colors.text),
    );
  }
}

class _InfoCard extends StatelessWidget {
  const _InfoCard({required this.rows});

  final List<_InfoRow> rows;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        children: [
          for (var i = 0; i < rows.length; i++) ...[
            if (i > 0) Divider(height: 1, color: colors.border),
            rows[i],
          ],
        ],
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value, this.leading});

  final String label;
  final String value;
  final IconData? leading;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      child: Row(
        children: [
          if (leading != null) ...[
            Icon(leading, size: 18, color: colors.textFaint),
            const SizedBox(width: 12),
          ],
          if (label.isNotEmpty) ...[
            Text(
              label,
              style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim),
            ),
            const SizedBox(width: 12),
          ],
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.end,
              style: TextStyle(
                  fontSize: 14.5,
                  fontWeight: FontWeight.w700,
                  color: colors.text),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlayerIdEditor extends StatelessWidget {
  const _PlayerIdEditor({
    required this.controller,
    required this.ids,
    required this.onAdd,
    required this.onRemove,
  });

  final TextEditingController controller;
  final List<String> ids;
  final VoidCallback onAdd;
  final ValueChanged<String> onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: controller,
                onSubmitted: (_) => onAdd(),
                textInputAction: TextInputAction.done,
                style: TextStyle(fontSize: 15, color: colors.text),
                cursorColor: AppTokens.brand1,
                decoration: InputDecoration(
                  filled: true,
                  fillColor: colors.surface,
                  hintText: l10n.playerIdHint,
                  hintStyle: TextStyle(color: colors.textFaint),
                  contentPadding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 14),
                  border: border,
                  enabledBorder: border,
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(AppTokens.rMd),
                    borderSide:
                        const BorderSide(color: AppTokens.brand1, width: 1.6),
                  ),
                ),
              ),
            ),
            const SizedBox(width: 10),
            FilledButton(
              onPressed: onAdd,
              style: FilledButton.styleFrom(
                backgroundColor: AppTokens.cta,
                foregroundColor: Colors.white,
                minimumSize: const Size(72, 50),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTokens.rMd),
                ),
              ),
              child: Text(l10n.addPlayerId),
            ),
          ],
        ),
        if (ids.isNotEmpty) ...[
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final id in ids)
                Chip(
                  label: Text(id),
                  labelStyle: TextStyle(
                      fontSize: 13.5,
                      fontWeight: FontWeight.w700,
                      color: colors.text),
                  backgroundColor: colors.surface,
                  side: BorderSide(color: colors.border),
                  deleteIcon: const Icon(Icons.close_rounded, size: 16),
                  onDeleted: () => onRemove(id),
                ),
            ],
          ),
        ],
      ],
    );
  }
}

class _SaveBar extends StatelessWidget {
  const _SaveBar({
    required this.label,
    required this.submitting,
    required this.colors,
    required this.onTap,
  });

  final String label;
  final bool submitting;
  final AppColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: submitting ? null : onTap,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.5),
          foregroundColor: Colors.white,
          disabledForegroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle: const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: submitting
            ? const SizedBox(
                width: 22,
                height: 22,
                child: CircularProgressIndicator(
                    strokeWidth: 2.5, color: Colors.white),
              )
            : Text(label),
      ),
    );
  }
}
