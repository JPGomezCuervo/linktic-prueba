<script setup lang="ts">
import type { ApiError } from "@composables/useApi";
import { isEmailValid, validateName, validatePassword } from "@composables/useValidation";
import {
	NAlert,
	NButton,
	NCard,
	NForm,
	NFormItem,
	NInput,
	NModal,
	NSpace,
	useMessage,
} from "naive-ui";
import { computed, ref } from "vue";

import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();
const message = useMessage();

const showEditModal = ref(false);
const isSaving = ref(false);
const errorMessage = ref("");

const editName = ref("");
const editEmail = ref("");
const newPassword = ref("");
const currentPassword = ref("");
const showPasswordHelp = ref(false);

const shouldChangePassword = computed(() => newPassword.value.trim() !== "");

function openEditModal(): void {
	editName.value = authStore.profile?.name ?? "";
	editEmail.value = authStore.profile?.email ?? "";
	newPassword.value = "";
	currentPassword.value = "";
	showPasswordHelp.value = false;
	errorMessage.value = "";
	showEditModal.value = true;
}

function closeEditModal(): void {
	showEditModal.value = false;
	editName.value = "";
	editEmail.value = "";
	newPassword.value = "";
	currentPassword.value = "";
	showPasswordHelp.value = false;
	errorMessage.value = "";
}

function togglePasswordHelp(): void {
	showPasswordHelp.value = !showPasswordHelp.value;
}

async function handleSave(): Promise<void> {
	errorMessage.value = "";

	const nameError = validateName(editName.value);
	if (nameError) {
		errorMessage.value = nameError;
		return;
	}

	if (!isEmailValid(editEmail.value.trim())) {
		errorMessage.value = "Please enter a valid email address.";
		return;
	}

	if (shouldChangePassword.value) {
		const passwordError = validatePassword(newPassword.value);
		if (passwordError) {
			errorMessage.value = passwordError;
			return;
		}

		if (currentPassword.value.trim() === "") {
			errorMessage.value = "Current password is required to change password.";
			return;
		}
	}

	isSaving.value = true;
	try {
		await authStore.updateProfile({
			name: editName.value.trim(),
			email: editEmail.value.trim(),
			password: shouldChangePassword.value ? newPassword.value : undefined,
			currentPassword: shouldChangePassword.value ? currentPassword.value : undefined,
		});

		message.success("Profile updated successfully.");
		closeEditModal();
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Failed to update profile.";
	} finally {
		isSaving.value = false;
	}
}
</script>

<template>
	<main class="space-y-5">
		<header class="flex items-center justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold text-slate-900">My Profile</h1>
				<p class="text-slate-600">Basic account information.</p>
			</div>
			<n-button type="primary" @click="openEditModal">Edit profile</n-button>
		</header>

		<n-card :bordered="false" class="rounded-2xl shadow-lg shadow-slate-900/8">
			<div class="space-y-2">
				<p class="text-sm text-slate-500">Name</p>
				<p class="text-lg font-medium text-slate-900">{{ authStore.profile?.name ?? "-" }}</p>
			</div>
			<div class="mt-6 space-y-2">
				<p class="text-sm text-slate-500">Email</p>
				<p class="text-lg font-medium text-slate-900">{{ authStore.profile?.email ?? "-" }}</p>
			</div>
		</n-card>

		<n-modal
			v-model:show="showEditModal"
			preset="card"
			title="Edit profile"
			class="w-full max-w-[520px] rounded-xl"
			:mask-closable="false"
		>
			<n-alert
				v-if="errorMessage"
				type="error"
				:show-icon="false"
				closable
				@close="errorMessage = ''"
				class="mb-4"
			>
				{{ errorMessage }}
			</n-alert>
			<n-form @submit.prevent="handleSave">
				<n-space vertical>
					<n-form-item label="Name">
						<n-input v-model:value="editName" placeholder="Jane Doe" :maxlength="100" show-count />
					</n-form-item>
					<n-form-item label="Email">
						<n-input v-model:value="editEmail" placeholder="you@email.com" />
					</n-form-item>
					<n-form-item label="New password (optional)">
						<n-input
							v-model:value="newPassword"
							type="password"
							show-password-on="click"
							placeholder="At least 8 characters"
						/>
					</n-form-item>
					<div class="-mt-3">
						<n-button text type="primary" @click="togglePasswordHelp">
							{{ showPasswordHelp ? "Hide password help" : "Show password help" }}
						</n-button>
						<div
							v-if="showPasswordHelp"
							class="mt-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700"
						>
							<p>- Leave "New password" empty to keep your current password.</p>
							<p>- New password must have at least 8 characters.</p>
							<p>- You must enter your current password to confirm password changes.</p>
						</div>
					</div>
					<n-form-item label="Current password" v-if="shouldChangePassword">
						<n-input
							v-model:value="currentPassword"
							type="password"
							show-password-on="click"
							placeholder="Required to change password"
						/>
					</n-form-item>
					<n-space justify="end">
						<n-button @click="closeEditModal">Cancel</n-button>
						<n-button type="primary" attr-type="submit" :loading="isSaving">Save changes</n-button>
					</n-space>
				</n-space>
			</n-form>
		</n-modal>
	</main>
</template>