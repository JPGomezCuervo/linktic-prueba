<script setup lang="ts">
defineProps<{
	title: string;
	subtitle: string;
}>();
</script>

<template>
	<div class="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-5 py-12 text-center">
		<p class="text-base font-medium text-slate-700">{{ title }}</p>
		<p class="mt-2 text-sm text-slate-500">{{ subtitle }}</p>
	</div>
</template>