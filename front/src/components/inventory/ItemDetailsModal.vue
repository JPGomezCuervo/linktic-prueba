<script setup lang="ts">
import { currencyFromCents, formatTimestamp } from "@composables/useFormatters";
import type { InventoryAuditEntry, InventoryItemDetails } from "@composables/useInventory";
import { NAlert, NButton, NModal } from "naive-ui";
import { computed } from "vue";

const props = defineProps<{
	show: boolean;
	isLoading: boolean;
	errorMessage: string;
	details: InventoryItemDetails | null;
}>();

const emit = defineEmits<{
	(event: "update:show", value: boolean): void;
	(event: "close"): void;
}>();

const detailHistory = computed(() => props.details?.history ?? []);

function formatAuditOperation(operation: InventoryAuditEntry["operation"]): string {
	if (operation === "create") {
		return "Created";
	}

	if (operation === "update") {
		return "Updated";
	}

	if (operation === "restore") {
		return "Restored";
	}

	if (operation === "restock") {
		return "Restocked";
	}

	return "Deleted";
}

function formatAuditValue(field: string, value: unknown): string {
	if (value === null || value === undefined) {
		return "-";
	}

	if (field === "price" && typeof value === "number") {
		return currencyFromCents(value);
	}

	if (typeof value === "boolean") {
		return value ? "true" : "false";
	}

	return String(value);
}

function getAuditChangeEntries(
	entry: InventoryAuditEntry,
): [string, { from: unknown; to: unknown }][] {
	return Object.entries(entry.changes ?? {});
}

function handleClose(): void {
	emit("update:show", false);
	emit("close");
}
</script>

<template>
	<n-modal
		:show="show"
		@update:show="(value) => emit('update:show', value)"
		preset="card"
		title="Item details"
		class="w-full max-w-[760px] rounded-xl"
		:mask-closable="true"
	>
		<div v-if="isLoading" class="py-6 text-center text-slate-600">Loading details...</div>
		<div v-else class="space-y-5">
			<n-alert v-if="errorMessage" type="error" :show-icon="false" closable class="mb-4">
				{{ errorMessage }}
			</n-alert>

			<template v-if="details">
				<div class="grid gap-3 md:grid-cols-2">
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Name</p>
						<p class="mt-1 text-base font-medium text-slate-900">{{ details.item.name }}</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Units</p>
						<p class="mt-1 text-base font-medium text-slate-900">{{ details.item.units }}</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Price</p>
						<p class="mt-1 text-base font-medium text-slate-900">
							{{ currencyFromCents(details.item.price) }}
						</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Status</p>
						<p class="mt-1 text-base font-medium text-slate-900">
							{{ details.item.deleted ? "Deleted" : "Active" }}
						</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4 md:col-span-2">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Item ID</p>
						<p class="mt-1 text-sm break-all text-slate-900">{{ details.item.id }}</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Created at</p>
						<p class="mt-1 text-sm text-slate-900">{{ formatTimestamp(details.item.createdAt) }}</p>
					</div>
					<div class="rounded-xl border border-slate-200 p-4">
						<p class="text-xs tracking-wide text-slate-500 uppercase">Updated at</p>
						<p class="mt-1 text-sm text-slate-900">{{ formatTimestamp(details.item.updatedAt) }}</p>
					</div>
				</div>

				<div>
					<h3 class="text-base font-semibold text-slate-900">Change history</h3>
					<div
						v-if="detailHistory.length === 0"
						class="mt-3 rounded-xl border border-dashed border-slate-300 bg-slate-50 p-4 text-sm text-slate-600"
					>
						No changes recorded yet.
					</div>
					<div v-else class="mt-3 space-y-3">
						<div
							v-for="entry in detailHistory"
							:key="entry.id"
							class="rounded-xl border border-slate-200 p-4"
						>
							<div class="flex flex-wrap items-center justify-between gap-2">
								<p class="text-sm font-semibold text-slate-900">
									{{ formatAuditOperation(entry.operation) }}
								</p>
								<p class="text-xs text-slate-500">{{ formatTimestamp(entry.createdAt) }}</p>
							</div>
							<p class="mt-1 text-xs text-slate-600">
								By {{ entry.actorName }} ({{ entry.actorEmail }})
							</p>
							<div class="mt-3 space-y-2">
								<div
									v-for="[field, change] in getAuditChangeEntries(entry)"
									:key="`${entry.id}-${field}`"
									class="rounded-lg bg-slate-50 px-3 py-2 text-sm"
								>
									<p class="font-medium text-slate-800">{{ field }}</p>
									<p class="text-slate-600">from: {{ formatAuditValue(field, change.from) }}</p>
									<p class="text-slate-600">to: {{ formatAuditValue(field, change.to) }}</p>
								</div>
							</div>
						</div>
					</div>
				</div>
			</template>
		</div>
		<template #footer>
			<div class="flex justify-end">
				<n-button @click="handleClose">Close</n-button>
			</div>
		</template>
	</n-modal>
</template>