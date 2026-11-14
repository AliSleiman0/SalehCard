import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/format/date_time.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/locale/locale_controller.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../checkout/presentation/providers.dart';
import '../../data/complaint_repository.dart';
import '../../domain/complaint.dart';
import '../complaint_labels.dart';

/// New complaint form: category, optional order (their recent orders),
/// subject, description and an optional photo. Submit stays disabled while a
/// photo upload is in flight.
class NewComplaintScreen extends ConsumerStatefulWidget {
  const NewComplaintScreen({super.key, this.orderId, this.category});

  /// Pre-filled from an order's "Report a problem" button.
  final String? orderId;
  final String? category;

  @override
  ConsumerState<NewComplaintScreen> createState() => _NewComplaintScreenState();
}

class _NewComplaintScreenState extends ConsumerState<NewComplaintScreen> {
  final _subject = TextEditingController();
  final _message = TextEditingController();
  late String _category;
  String _orderId = '';
  String? _photoUrl;
  bool _uploading = false;
  bool _submitting = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _orderId = widget.orderId ?? '';
    final preset = widget.category;
    _category = preset != null && complaintCategories.contains(preset)
        ? preset
        : (_orderId.isNotEmpty ? 'order' : 'other');
  }

  @override
  void dispose() {
    _subject.dispose();
    _message.dispose();
    super.dispose();
  }

  Future<void> _pickPhoto() async {
    FocusScope.of(context).unfocus();
    final l10n = AppLocalizations.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final picked = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      imageQuality: 85,
      maxWidth: 2000,
    );
    if (picked == null || !mounted) return;
    setState(() => _uploading = true);
    try {
      final url = await ref
          .read(complaintRepositoryProvider)
          .uploadPhoto(picked.path);
      if (mounted) setState(() => _photoUrl = url);
    } catch (_) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.uploadFailed)));
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    final subject = _subject.text.trim();
    final message = _message.text.trim();
    if (subject.isEmpty) {
      setState(() => _error = l10n.complaintErrSubject);
      return;
    }
    if (message.isEmpty) {
      setState(() => _error = l10n.complaintErrMessage);
      return;
    }
    setState(() {
      _error = null;
      _submitting = true;
    });
    try {
      final c = await ref
          .read(complaintRepositoryProvider)
          .create(
            category: _category,
            subject: subject,
            message: message,
            orderId: _category == 'order' ? _orderId : null,
            photoUrl: _photoUrl,
          );
      ref.invalidate(complaintsProvider);
      if (mounted) context.pushReplacement('/complaints/${c.id}');
    } catch (e) {
      final failure = mapError(e);
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _error = failure is ServerFailure && failure.code == 'TOO_MANY_OPEN'
            ? l10n.complaintTooManyOpen
            : (failure.message.isNotEmpty
                  ? failure.message
                  : l10n.complaintSendFailed);
      });
    }
  }

  InputDecoration _decoration(String hint) => InputDecoration(
    hintText: hint,
    border: OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
    ),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final localeCode = ref.watch(localeControllerProvider).languageCode;
    final orders = ref.watch(ordersProvider).value ?? const [];
    final recent = orders.take(20).toList();

    Widget label(String text) => Padding(
      padding: const EdgeInsets.only(bottom: 8, top: 16),
      child: Text(
        text,
        style: TextStyle(fontWeight: FontWeight.w700, color: colors.text),
      ),
    );

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.complaintNewTitle),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
        children: [
          Text(l10n.complaintNewSub, style: TextStyle(color: colors.textDim)),
          label(l10n.complaintCategory),
          DropdownButtonFormField<String>(
            initialValue: _category,
            decoration: _decoration(''),
            items: [
              for (final c in complaintCategories)
                DropdownMenuItem(
                  value: c,
                  child: Text(complaintCategoryLabel(c, l10n)),
                ),
            ],
            onChanged: (v) => setState(() => _category = v ?? 'other'),
          ),
          if (_category == 'order') ...[
            label(l10n.complaintOrderOptional),
            DropdownButtonFormField<String>(
              initialValue: _orderId,
              isExpanded: true,
              decoration: _decoration(''),
              items: [
                DropdownMenuItem(value: '', child: Text(l10n.complaintNoOrder)),
                if (_orderId.isNotEmpty && !recent.any((o) => o.id == _orderId))
                  DropdownMenuItem(
                    value: _orderId,
                    child: Text(
                      '#${_orderId.substring(_orderId.length > 8 ? _orderId.length - 8 : 0)}',
                    ),
                  ),
                for (final o in recent)
                  DropdownMenuItem(
                    value: o.id,
                    child: Text(
                      '#${o.reference} · ${o.items.isEmpty ? '' : o.items.first.title.resolve(localeCode)} · ${formatDateTime(o.createdAt)}',
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
              ],
              onChanged: (v) => setState(() => _orderId = v ?? ''),
            ),
          ],
          label(l10n.complaintSubject),
          TextField(
            controller: _subject,
            maxLength: 120,
            decoration: _decoration(l10n.complaintSubjectHint),
          ),
          label(l10n.complaintMessage),
          TextField(
            controller: _message,
            maxLength: 2000,
            minLines: 4,
            maxLines: 8,
            decoration: _decoration(l10n.complaintMessageHint),
          ),
          label(l10n.complaintPhotoOptional),
          if (_photoUrl != null)
            Row(
              children: [
                ClipRRect(
                  borderRadius: BorderRadius.circular(AppTokens.rSm),
                  child: Image.network(
                    _photoUrl!,
                    width: 84,
                    height: 84,
                    fit: BoxFit.cover,
                  ),
                ),
                const SizedBox(width: 12),
                TextButton(
                  onPressed: () => setState(() => _photoUrl = null),
                  child: Text(l10n.complaintRemovePhoto),
                ),
              ],
            )
          else
            OutlinedButton.icon(
              onPressed: _uploading ? null : _pickPhoto,
              icon: _uploading
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.add_photo_alternate_outlined),
              label: Text(l10n.complaintAttachPhoto),
            ),
          if (_error != null) ...[
            const SizedBox(height: 16),
            Text(_error!, style: const TextStyle(color: AppTokens.danger)),
          ],
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _uploading || _submitting ? null : _submit,
            child: _submitting
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text(l10n.complaintSubmit),
          ),
        ],
      ),
    );
  }
}
