<script setup lang="ts">
import type { ApiError } from "@composables/useApi";
import { isEmailValid, validateName, validatePassword } from "@composables/useValidation";
import { NAlert, NButton, NCard, NForm, NFormItem, NInput, NModal, NSpace } from "naive-ui";
import { ref } from "vue";
import { useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const authStore = useAuthStore();

const email = ref("");
const password = ref("");
const name = ref("");

const showSignupModal = ref(false);
const isLoginLoading = ref(false);
const isSignupLoading = ref(false);

const errorMessage = ref("");
const successMessage = ref("");

function resetFeedback(): void {
	errorMessage.value = "";
	successMessage.value = "";
}

function openSignup(): void {
	name.value = "";
	password.value = "";
	showSignupModal.value = true;
	resetFeedback();
}

function forgotPassword(): void {
	resetFeedback();
	successMessage.value = "Forgot password is not implemented yet.";
}

async function handleLogin(): Promise<void> {
	resetFeedback();

	if (!isEmailValid(email.value)) {
		errorMessage.value = "Please enter a valid email address.";
		return;
	}

	if (password.value.trim() === "") {
		errorMessage.value = "Password is required.";
		return;
	}

	isLoginLoading.value = true;
	try {
		await authStore.login(email.value.trim(), password.value);
		await router.push("/app");
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Login failed.";
	} finally {
		isLoginLoading.value = false;
	}
}

async function handleSignup(): Promise<void> {
	resetFeedback();

	const nameError = validateName(name.value);
	if (nameError) {
		errorMessage.value = nameError;
		return;
	}

	if (!isEmailValid(email.value)) {
		errorMessage.value = "Please enter a valid email address.";
		return;
	}

	const passwordError = validatePassword(password.value);
	if (passwordError) {
		errorMessage.value = passwordError;
		return;
	}

	isSignupLoading.value = true;
	try {
		await authStore.signup(name.value.trim(), email.value.trim(), password.value);
		showSignupModal.value = false;
		successMessage.value = "Account created. You can log in now.";
	} catch (error) {
		const apiError = error as ApiError;
		errorMessage.value = apiError.message || "Signup failed.";
	} finally {
		isSignupLoading.value = false;
	}
}
</script>

<template>
	<n-card
		class="w-full max-w-[460px] rounded-2xl bg-white/95 shadow-2xl shadow-slate-900/12 backdrop-blur-sm"
		title="Welcome back"
		:bordered="false"
	>
		<n-space vertical size="large">
			<n-alert
				v-if="errorMessage"
				type="error"
				:show-icon="false"
				closable
				@close="errorMessage = ''"
			>
				{{ errorMessage }}
			</n-alert>

			<n-alert
				v-if="successMessage"
				type="success"
				:show-icon="false"
				closable
				@close="successMessage = ''"
			>
				{{ successMessage }}
			</n-alert>

			<n-form @submit.prevent="handleLogin">
				<n-form-item label="Email">
					<n-input v-model:value="email" type="text" placeholder="you@company.com" />
				</n-form-item>

				<n-form-item label="Password">
					<n-input
						v-model:value="password"
						type="password"
						show-password-on="click"
						placeholder="********"
					/>
				</n-form-item>

				<n-space vertical size="small">
					<n-button type="primary" attr-type="submit" :loading="isLoginLoading" block>
						Login
					</n-button>
					<n-button quaternary type="default" @click="openSignup" block> Signup </n-button>
					<n-button text type="default" @click="forgotPassword" block> Forgot password </n-button>
				</n-space>
			</n-form>
		</n-space>
	</n-card>

	<n-modal
		v-model:show="showSignupModal"
		preset="card"
		title="Create account"
		class="w-full max-w-[520px] rounded-xl"
		:mask-closable="false"
	>
		<n-alert
			v-if="errorMessage"
			type="error"
			:show-icon="false"
			closable
			@close="errorMessage = ''"
		>
			{{ errorMessage }}
		</n-alert>
		<n-form @submit.prevent="handleSignup">
			<n-space vertical>
				<n-form-item label="Name">
					<n-input v-model:value="name" placeholder="Jane Doe" :maxlength="100" show-count />
				</n-form-item>
				<n-form-item label="Email">
					<n-input v-model:value="email" type="text" placeholder="you@email.com" />
				</n-form-item>
				<n-form-item label="Password">
					<n-input
						v-model:value="password"
						type="password"
						show-password-on="click"
						placeholder="At least 8 characters"
					/>
				</n-form-item>

				<n-space justify="end">
					<n-button @click="showSignupModal = false">Cancel</n-button>
					<n-button type="primary" attr-type="submit" :loading="isSignupLoading"
						>Create account</n-button
					>
				</n-space>
			</n-space>
		</n-form>
	</n-modal>
</template>