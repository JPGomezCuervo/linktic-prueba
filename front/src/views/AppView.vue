<script setup lang="ts">
import { NButton } from "naive-ui";
import { useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();
const router = useRouter();

async function handleLogout(): Promise<void> {
	await authStore.logout();
	await router.push("/login");
}
</script>

<template>
	<main class="grid min-h-screen place-items-center p-6">
		<section class="w-full max-w-xl space-y-4 rounded-2xl bg-white p-8 text-center shadow-xl">
			<h1 class="text-4xl font-semibold text-slate-900">welcome</h1>
			<p class="text-slate-600" v-if="authStore.profile">
				Signed in as {{ authStore.profile.email }}
			</p>
			<n-button type="primary" @click="handleLogout">Logout</n-button>
		</section>
	</main>
</template>