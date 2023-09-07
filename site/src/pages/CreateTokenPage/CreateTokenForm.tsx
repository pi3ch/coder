import { FC, useState, useEffect } from "react";
import {
  FormFields,
  FormSection,
  FormFooter,
  HorizontalForm,
} from "components/Form/Form";
import makeStyles from "@mui/styles/makeStyles";
import { useTranslation } from "react-i18next";
import { onChangeTrimmed, getFormHelpers } from "utils/formUtils";
import TextField from "@mui/material/TextField";
import MenuItem from "@mui/material/MenuItem";
import {
  NANO_HOUR,
  CreateTokenData,
  determineDefaultLtValue,
  filterByMaxTokenLifetime,
  customLifetimeDay,
} from "./utils";
import { FormikContextType } from "formik";
import dayjs from "dayjs";
import { useNavigate } from "react-router-dom";
import { Stack } from "components/Stack/Stack";

interface CreateTokenFormProps {
  form: FormikContextType<CreateTokenData>;
  maxTokenLifetime?: number;
  formError: unknown;
  setFormError: (arg0: unknown) => void;
  isCreating: boolean;
  creationFailed: boolean;
}

export const CreateTokenForm: FC<CreateTokenFormProps> = ({
  form,
  maxTokenLifetime,
  formError,
  setFormError,
  isCreating,
  creationFailed,
}) => {
  const styles = useStyles();
  const { t } = useTranslation("tokensPage");
  const navigate = useNavigate();

  const [expDays, setExpDays] = useState<number>(1);
  const [lifetimeDays, setLifetimeDays] = useState<number | string>(
    determineDefaultLtValue(maxTokenLifetime),
  );

  useEffect(() => {
    if (lifetimeDays !== "custom") {
      void form.setFieldValue("lifetime", lifetimeDays);
    } else {
      void form.setFieldValue("lifetime", expDays);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- adding form will cause an infinite loop
  }, [lifetimeDays, expDays]);

  const getFieldHelpers = getFormHelpers<CreateTokenData>(form, formError);

  return (
    <HorizontalForm onSubmit={form.handleSubmit}>
      <FormSection
        title="Name"
        description="What is this token for?"
        classes={{ sectionInfo: styles.formSectionInfo }}
      >
        <FormFields>
          <TextField
            {...getFieldHelpers("name")}
            label={t("createToken.fields.name")}
            required
            onChange={onChangeTrimmed(form, () => setFormError(undefined))}
            autoFocus
            fullWidth
          />
        </FormFields>
      </FormSection>
      <FormSection
        title="Expiration"
        description={
          form.values.lifetime
            ? `The token will expire on ${dayjs()
                .add(form.values.lifetime, "days")
                .utc()
                .format("MMMM DD, YYYY")}`
            : "Please set a token expiration."
        }
        classes={{ sectionInfo: styles.formSectionInfo }}
      >
        <FormFields>
          <Stack direction="row">
            <TextField
              select
              label={t("createToken.fields.lifetime")}
              required
              defaultValue={determineDefaultLtValue(maxTokenLifetime)}
              onChange={(event) => {
                void setLifetimeDays(event.target.value);
              }}
              fullWidth
            >
              {filterByMaxTokenLifetime(maxTokenLifetime).map((lt) => (
                <MenuItem key={lt.label} value={lt.value}>
                  {lt.label}
                </MenuItem>
              ))}
              <MenuItem
                key={customLifetimeDay.label}
                value={customLifetimeDay.value}
              >
                {customLifetimeDay.label}
              </MenuItem>
            </TextField>

            {lifetimeDays === "custom" && (
              <TextField
                type="date"
                label={t("createToken.lifetimeSection.expiresOn")}
                defaultValue={dayjs().add(expDays, "day").format("YYYY-MM-DD")}
                onChange={(event) => {
                  const lt = Math.ceil(
                    dayjs(event.target.value).diff(dayjs(), "day", true),
                  );
                  setExpDays(lt);
                }}
                inputProps={{
                  min: dayjs().add(1, "day").format("YYYY-MM-DD"),
                  max: maxTokenLifetime
                    ? dayjs()
                        .add(maxTokenLifetime / NANO_HOUR / 24, "day")
                        .format("YYYY-MM-DD")
                    : undefined,
                  required: true,
                }}
                fullWidth
                InputLabelProps={{
                  required: true,
                }}
              />
            )}
          </Stack>
        </FormFields>
      </FormSection>
      <FormFooter
        onCancel={() => navigate("/settings/tokens")}
        isLoading={isCreating}
        submitLabel={creationFailed ? "Retry" : "Create token"}
      />
    </HorizontalForm>
  );
};

const useStyles = makeStyles(() => ({
  formSectionInfo: {
    minWidth: "300px",
  },
}));
