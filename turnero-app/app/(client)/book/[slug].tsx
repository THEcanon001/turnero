import { useState } from "react";
import {
  View,
  Text,
  ScrollView,
  ActivityIndicator,
  Alert,
} from "react-native";
import { useLocalSearchParams, useRouter, Stack } from "expo-router";
import {
  useProviderProfile,
  useSlots,
  useCreateAppointment,
} from "../../../hooks/useApi";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { Card } from "../../../components/ui/Card";
import { DayPicker } from "../../../components/calendar/DayPicker";
import { SlotGrid } from "../../../components/calendar/SlotGrid";
import { EmptyState } from "../../../components/ui/EmptyState";
import { openWhatsApp } from "../../../lib/whatsapp";
import { getApiErrorMessage } from "../../../lib/api";
import type { Service, Employee } from "../../../types/api";

type BookingStep =
  | "service"
  | "employee"
  | "date"
  | "time"
  | "details"
  | "confirm"
  | "done";

export default function BookingFlowScreen() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const router = useRouter();
  const { data: provider, isLoading: providerLoading } = useProviderProfile(
    slug ?? ""
  );
  const createAppointment = useCreateAppointment();

  const [step, setStep] = useState<BookingStep>("service");
  const [selectedService, setSelectedService] = useState<Service | null>(null);
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(
    null
  );
  // For "any available" we use a placeholder employee id
  const [anyEmployee, setAnyEmployee] = useState(false);
  const [selectedDate, setSelectedDate] = useState("");
  const [selectedSlot, setSelectedSlot] = useState<string | null>(null);
  const [clientName, setClientName] = useState("");
  const [clientPhone, setClientPhone] = useState("");
  const [notes, setNotes] = useState("");
  const [whatsappLink, setWhatsappLink] = useState("");

  // For slots, we need a real employee ID. If "any available", we'll use the
  // first employee from provider response. The backend handles "any" logic.
  const employeeIdForSlots = selectedEmployee?.id ?? "";

  const {
    data: slots,
    isLoading: slotsLoading,
  } = useSlots(slug ?? "", employeeIdForSlots, selectedDate);

  if (providerLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Stack.Screen options={{ headerShown: true, title: "Agendar turno" }} />
        <ActivityIndicator size="large" color="#6366F1" />
      </View>
    );
  }

  if (!provider) {
    return (
      <View className="flex-1">
        <Stack.Screen options={{ headerShown: true, title: "Error" }} />
        <EmptyState title="No encontrado" />
      </View>
    );
  }

  const activeServices = provider.services.filter((s) => s.is_active);

  const handleConfirm = async () => {
    if (!clientName.trim() || !clientPhone.trim()) {
      Alert.alert("Error", "Completá tu nombre y teléfono.");
      return;
    }

    try {
      const result = await createAppointment.mutateAsync({
        provider_slug: slug!,
        employee_id: anyEmployee ? null : selectedEmployee?.id ?? null,
        service_id: selectedService?.id ?? null,
        client_name: clientName.trim(),
        client_phone: clientPhone.trim(),
        date: selectedDate,
        start_time: selectedSlot!,
        notes: notes.trim() || null,
      });
      setWhatsappLink(result.whatsapp_link);
      setStep("done");
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    }
  };

  const stepTitle: Record<BookingStep, string> = {
    service: "1. Elegí un servicio",
    employee: "2. Elegí un profesional",
    date: "3. Elegí una fecha",
    time: "4. Elegí un horario",
    details: "5. Tus datos",
    confirm: "6. Confirmar turno",
    done: "Turno confirmado",
  };

  return (
    <View className="flex-1">
      <Stack.Screen
        options={{ headerShown: true, title: "Agendar turno" }}
      />
      <ScrollView
        contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
        keyboardShouldPersistTaps="handled"
      >
        <Text className="text-xl font-bold text-gray-900 mb-4">
          {stepTitle[step]}
        </Text>

        {/* Step 1: Select service */}
        {step === "service" && (
          <View className="gap-3">
            {activeServices.map((service) => (
              <Card
                key={service.id}
                className={
                  selectedService?.id === service.id
                    ? "border-primary-600 border-2"
                    : ""
                }
              >
                <Button
                  title={`${service.name} (${service.duration_minutes} min)`}
                  variant={
                    selectedService?.id === service.id
                      ? "primary"
                      : "secondary"
                  }
                  onPress={() => {
                    setSelectedService(service);
                    setStep("employee");
                  }}
                />
              </Card>
            ))}
            <Button
              title="Sin servicio específico"
              variant="outline"
              onPress={() => {
                setSelectedService(null);
                setStep("employee");
              }}
            />
          </View>
        )}

        {/* Step 2: Select employee (or any) */}
        {step === "employee" && (
          <View className="gap-3">
            <Button
              title="Cualquier profesional disponible"
              variant="outline"
              onPress={() => {
                setAnyEmployee(true);
                setSelectedEmployee(null);
                setStep("date");
              }}
            />
            <Text className="text-sm text-gray-400 text-center my-2">
              La selección de profesional específico estará disponible próximamente.
            </Text>
            <Button
              title="Volver"
              variant="secondary"
              onPress={() => setStep("service")}
            />
          </View>
        )}

        {/* Step 3: Select date */}
        {step === "date" && (
          <View>
            <DayPicker
              selectedDate={selectedDate}
              onSelectDate={(date) => {
                setSelectedDate(date);
                setSelectedSlot(null);
                setStep("time");
              }}
            />
            <Button
              title="Volver"
              variant="secondary"
              onPress={() => setStep("employee")}
              className="mt-4"
            />
          </View>
        )}

        {/* Step 4: Select time slot */}
        {step === "time" && (
          <View>
            <DayPicker
              selectedDate={selectedDate}
              onSelectDate={(date) => {
                setSelectedDate(date);
                setSelectedSlot(null);
              }}
            />
            <View className="mt-4">
              <SlotGrid
                slots={slots ?? []}
                selectedSlot={selectedSlot}
                onSelectSlot={setSelectedSlot}
                loading={slotsLoading}
              />
            </View>
            {selectedSlot && (
              <Button
                title="Continuar"
                onPress={() => setStep("details")}
                className="mt-4"
              />
            )}
            <Button
              title="Volver"
              variant="secondary"
              onPress={() => setStep("date")}
              className="mt-2"
            />
          </View>
        )}

        {/* Step 5: Client details */}
        {step === "details" && (
          <View>
            <Input
              label="Tu nombre"
              placeholder="Nombre completo"
              value={clientName}
              onChangeText={setClientName}
            />
            <Input
              label="Tu teléfono"
              placeholder="+54 11 1234 5678"
              value={clientPhone}
              onChangeText={setClientPhone}
              keyboardType="phone-pad"
            />
            <Input
              label="Notas (opcional)"
              placeholder="Alguna indicación especial..."
              value={notes}
              onChangeText={setNotes}
              multiline
            />
            <Button
              title="Revisar turno"
              onPress={() => setStep("confirm")}
              disabled={!clientName.trim() || !clientPhone.trim()}
            />
            <Button
              title="Volver"
              variant="secondary"
              onPress={() => setStep("time")}
              className="mt-2"
            />
          </View>
        )}

        {/* Step 6: Confirm */}
        {step === "confirm" && (
          <View>
            <Card className="mb-4">
              <Text className="text-base font-semibold text-gray-800">
                Resumen del turno
              </Text>
              <View className="mt-3 gap-1">
                <Text className="text-sm text-gray-600">
                  Profesional: {provider.name}
                </Text>
                {selectedService && (
                  <Text className="text-sm text-gray-600">
                    Servicio: {selectedService.name}
                  </Text>
                )}
                <Text className="text-sm text-gray-600">
                  Fecha: {selectedDate}
                </Text>
                <Text className="text-sm text-gray-600">
                  Horario: {selectedSlot}
                </Text>
                <Text className="text-sm text-gray-600">
                  Nombre: {clientName}
                </Text>
                <Text className="text-sm text-gray-600">
                  Teléfono: {clientPhone}
                </Text>
                {notes ? (
                  <Text className="text-sm text-gray-600">
                    Notas: {notes}
                  </Text>
                ) : null}
              </View>
            </Card>
            <Button
              title="Confirmar turno"
              onPress={handleConfirm}
              loading={createAppointment.isPending}
            />
            <Button
              title="Volver"
              variant="secondary"
              onPress={() => setStep("details")}
              className="mt-2"
            />
          </View>
        )}

        {/* Done */}
        {step === "done" && (
          <View className="items-center">
            <Text className="text-2xl font-bold text-green-600 mb-4">
              Turno confirmado
            </Text>
            <Card className="mb-4 w-full">
              <Text className="text-base text-gray-700 text-center">
                Tu turno para el {selectedDate} a las {selectedSlot} fue
                confirmado exitosamente.
              </Text>
            </Card>
            {whatsappLink && (
              <Button
                title="Enviar por WhatsApp"
                variant="outline"
                onPress={() => openWhatsApp(provider.phone, "Turno confirmado")}
                className="mb-3 w-full"
              />
            )}
            <Button
              title="Volver al inicio"
              variant="secondary"
              onPress={() => router.replace("/(client)")}
              className="w-full"
            />
          </View>
        )}
      </ScrollView>
    </View>
  );
}
